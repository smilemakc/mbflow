package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ErrResourceUnavailable is returned (wrapped) when a resource attached to an
// execution cannot be loaded, is not owned by the execution's owner, is not
// active, or is a credential that cannot be decrypted.
var ErrResourceUnavailable = errors.New("resource unavailable")

// errCredentialsDisabled explains why credentials cannot be resolved when the
// server runs without an encryption key.
var errCredentialsDisabled = fmt.Errorf(
	"credential decryption is not available: the server has no encryption key configured (set %s)", crypto.KeyEnvVar)

// credentialDecrypter decrypts the values of a credential.
type credentialDecrypter interface {
	DecryptMap(data map[string]string) (map[string]string, error)
}

// credentialUsageRecorder records that a credential was used.
type credentialUsageRecorder interface {
	IncrementUsageCount(ctx context.Context, resourceID string) error
}

// reservedResourceKeys are resource fields that decrypted credential values
// never override; such values stay reachable under "fields".
var reservedResourceKeys = map[string]bool{
	"id": true, "name": true, "type": true, "access_type": true,
	"credential_type": true, "provider": true, "fields": true,
}

// SetCredentialDecryption enables credential resources in executions: their
// decrypted values become template fields ({{resource.<alias>.api_key}}) and
// are redacted from everything the execution persists or emits. usage, when
// not nil, records each use (usage_count / last_used_at); a failing update
// never fails the execution. Without it, executions that reference a
// credential's fields fail with a clear error.
func (em *ExecutionManager) SetCredentialDecryption(enc *crypto.EncryptionService, usage repository.CredentialsRepository) {
	em.credDecrypter = nil
	if enc != nil {
		em.credDecrypter = enc
	}
	em.credUsage = nil
	if usage != nil {
		em.credUsage = usage
	}
}

// loadResources loads the resources of an execution and checks that ownerID
// owns them (skipped when ownerID is empty). Decrypted credential values are
// added to secrets. workflow is used to tell whether a credential is actually
// referenced by a node, which decides whether a decryption problem is fatal.
func (em *ExecutionManager) loadResources(
	ctx context.Context,
	refs []models.WorkflowResource,
	ownerID string,
	workflow *models.Workflow,
	secrets *pkgengine.SecretRedactor,
) (map[string]any, error) {
	resourceMap := make(map[string]any, len(refs))
	if len(refs) == 0 {
		return resourceMap, nil
	}
	if em.resourceRepo == nil {
		return nil, fmt.Errorf("%w: resources are not supported (no resource repository configured)", ErrResourceUnavailable)
	}

	for _, wr := range refs {
		resource, err := em.resourceRepo.GetByID(ctx, wr.ResourceID)
		if err != nil || resource == nil {
			if err == nil {
				err = models.ErrResourceNotFound
			}
			return nil, fmt.Errorf("%w: failed to load resource %s (alias: %s): %w", ErrResourceUnavailable, wr.ResourceID, wr.Alias, err)
		}

		if ownerID != "" && resource.GetOwnerID() != ownerID {
			return nil, fmt.Errorf("%w: resource access denied: resource %s (alias: %s) owner does not match workflow owner",
				ErrResourceUnavailable, wr.ResourceID, wr.Alias)
		}

		if resource.GetStatus() != models.ResourceStatusActive {
			return nil, fmt.Errorf("%w: resource %s (alias: %s) is not active (status: %s)",
				ErrResourceUnavailable, wr.ResourceID, wr.Alias, resource.GetStatus())
		}

		accessType := wr.AccessType
		if accessType == "" {
			accessType = "read"
		}
		entry := map[string]any{
			"id":          resource.GetID(),
			"name":        resource.GetName(),
			"type":        string(resource.GetType()),
			"access_type": accessType,
		}

		if cred, ok := resource.(*models.CredentialsResource); ok {
			if err := em.addCredentialFields(ctx, entry, cred, wr.Alias, workflow, secrets); err != nil {
				return nil, err
			}
		}

		resourceMap[wr.Alias] = entry
	}

	return resourceMap, nil
}

// addCredentialFields decrypts cred and adds its values to entry. A credential
// that cannot be decrypted (no encryption key, wrong key, expired) fails the
// execution only when a node references it; otherwise only its metadata is
// exposed.
func (em *ExecutionManager) addCredentialFields(
	ctx context.Context,
	entry map[string]any,
	cred *models.CredentialsResource,
	alias string,
	workflow *models.Workflow,
	secrets *pkgengine.SecretRedactor,
) error {
	entry["credential_type"] = string(cred.CredentialType)
	if cred.Provider != "" {
		entry["provider"] = cred.Provider
	}

	values, err := em.decryptCredential(cred)
	if err != nil {
		if workflowReferencesResource(workflow, alias) {
			return fmt.Errorf("%w: credential %s (alias: %s) cannot be used: %w", ErrResourceUnavailable, cred.ID, alias, err)
		}
		slog.Warn("credential attached to execution is not usable; its fields are not available",
			"credential_id", cred.ID, "alias", alias, "error", err)
		return nil
	}

	fields := make(map[string]any, len(values))
	for k, v := range values {
		if !nonSecretCredentialField(cred.CredentialType, k) {
			secrets.Add(v)
		}
		fields[k] = v
		if !reservedResourceKeys[k] {
			entry[k] = v
		}
	}
	entry["fields"] = fields

	if em.credUsage != nil {
		if err := em.credUsage.IncrementUsageCount(ctx, cred.ID); err != nil {
			slog.Warn("failed to record credential usage", "credential_id", cred.ID, "error", err)
		}
	}
	return nil
}

// nonSecretCredentialField reports whether a credential field holds
// configuration rather than a secret (e.g. an OAuth2 token URL), so it is
// not redacted.
func nonSecretCredentialField(credType models.CredentialType, field string) bool {
	return credType == models.CredentialTypeOAuth2 && (field == "token_url" || field == "scopes")
}

func (em *ExecutionManager) decryptCredential(cred *models.CredentialsResource) (map[string]string, error) {
	if em.credDecrypter == nil {
		return nil, errCredentialsDisabled
	}
	if cred.IsExpired() {
		return nil, errors.New("credential has expired")
	}
	values, err := em.credDecrypter.DecryptMap(cred.EncryptedData)
	if err != nil {
		// The error names the field only; it never contains plaintext.
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}
	return values, nil
}

// workflowReferencesResource reports whether any node config of workflow
// mentions resource.<alias> (as used in templates).
func workflowReferencesResource(workflow *models.Workflow, alias string) bool {
	if workflow == nil || alias == "" {
		return false
	}
	pattern, err := regexp.Compile(`resource\.` + regexp.QuoteMeta(alias) + `(?:[^A-Za-z0-9_-]|$)`)
	if err != nil {
		return true
	}
	for _, node := range workflow.Nodes {
		if node == nil || len(node.Config) == 0 {
			continue
		}
		data, err := json.Marshal(node.Config)
		if err != nil {
			continue
		}
		if pattern.Match(data) {
			return true
		}
	}
	return false
}

// ephemeralResourceRefs returns the resources of an inline execution: the
// inline workflow's own resources plus credentialIDs. A credential listed in
// credentialIDs that the workflow also declares keeps the workflow's alias;
// otherwise its alias is the credential ID itself.
func ephemeralResourceRefs(workflow *models.Workflow, credentialIDs []string) []models.WorkflowResource {
	refs := make([]models.WorkflowResource, 0, len(workflow.Resources)+len(credentialIDs))
	declared := make(map[string]bool, len(workflow.Resources))
	for _, wr := range workflow.Resources {
		refs = append(refs, wr)
		declared[wr.ResourceID] = true
	}
	for _, id := range credentialIDs {
		if id == "" || declared[id] {
			continue
		}
		declared[id] = true
		refs = append(refs, models.WorkflowResource{ResourceID: id, Alias: id, AccessType: "read"})
	}
	return refs
}

// redactExecution removes the execution's secrets from everything that is
// persisted or returned: the execution's output and error, and each node's
// config, resolved config, input, output and error. The execution state keeps
// the real values.
func redactExecution(execution *models.Execution, execState *pkgengine.ExecutionState) {
	if execState == nil || execState.Secrets.Len() == 0 {
		return
	}
	secrets := execState.Secrets
	execution.Output = secrets.Map(execution.Output)
	execution.Error = secrets.String(execution.Error)
	for _, ne := range execution.NodeExecutions {
		if ne == nil {
			continue
		}
		ne.Config = secrets.Map(ne.Config)
		ne.ResolvedConfig = secrets.Map(ne.ResolvedConfig)
		ne.Input = secrets.Map(ne.Input)
		ne.Output = secrets.Map(ne.Output)
		ne.Error = secrets.String(ne.Error)
	}
}

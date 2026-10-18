package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/openapi"
)

const (
	// maxToolNameLength is the function name limit of OpenAI, Anthropic and Gemini.
	maxToolNameLength = 64
	// maxAPIToolResultBytes bounds the response body returned to the model:
	// the body becomes part of every following request of the conversation.
	maxAPIToolResultBytes = 16 * 1024
	// maxAPIToolErrorBytes bounds the body quoted in the error of a failed call.
	maxAPIToolErrorBytes = 2 * 1024
)

var toolNameInvalidChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

var toolArgGroups = []string{
	openapi.ToolArgPathParams, openapi.ToolArgQueryParams, openapi.ToolArgHeaders, openapi.ToolArgCookies,
}

// OperationTools returns a function definition for every operation that refs
// select. The clients must belong to the workflow owner, as for the openapi node.
func (e *APIOperationExecutor) OperationTools(ctx context.Context, refs []models.APIToolRef) ([]models.FunctionDefinition, error) {
	var defs []models.FunctionDefinition
	seen := make(map[string]bool)
	for _, ref := range refs {
		clientID := strings.TrimSpace(ref.ClientID)
		client, err := e.ownedClient(ctx, clientID)
		if err != nil {
			return nil, err
		}
		for _, operationID := range ref.OperationIDs {
			operationID = strings.TrimSpace(operationID)
			key := clientID + "\x00" + operationID
			if seen[key] {
				continue
			}
			seen[key] = true
			op := client.FindOperation(operationID)
			if op == nil {
				return nil, fmt.Errorf("%w: %q in API client %q; the specification may have changed",
					models.ErrAPIOperationNotFound, operationID, client.Name)
			}
			defs = append(defs, models.FunctionDefinition{
				Type:        models.FunctionTypeOpenAPI,
				Name:        toolName(op.ID),
				Description: openapi.ToolDescription(client, op),
				Parameters:  openapi.ToolParameters(client, op),
				ClientID:    clientID,
				OperationID: op.ID,
			})
		}
	}
	return defs, nil
}

// ExecuteOperation calls an operation with the arguments of a tool call and
// returns the status and the body of the response, cut to
// maxAPIToolResultBytes. A response with status 400 or above is an error, so
// the model sees it as a failed call and stop_on_tool_failure applies.
func (e *APIOperationExecutor) ExecuteOperation(ctx context.Context, clientID, operationID string, args map[string]any) (any, error) {
	client, op, err := e.operation(ctx, clientID, operationID)
	if err != nil {
		return nil, err
	}
	config, err := toolCallConfig(op, args)
	if err != nil {
		return nil, err
	}
	config["ignore_status_errors"] = true

	output, secrets, err := e.call(ctx, client, op, config, nil)
	if err != nil {
		return nil, err
	}
	result, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s %s: unexpected response %T", op.Method, op.Path, output)
	}

	status, _ := result["status"].(int)
	if _, binary := result["body_base64"]; binary {
		if status >= 400 {
			return nil, fmt.Errorf("HTTP %d", status)
		}
		return map[string]any{"status": status, "binary": true, "content_type": result["content_type"], "size": result["size"]}, nil
	}

	text, structured := bodyText(result["body"])
	text = maskSecrets(text, secrets)
	if status >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", status, truncateBytes(text, maxAPIToolErrorBytes))
	}

	content := map[string]any{"status": status}
	switch {
	case len(text) > maxAPIToolResultBytes:
		content["body"] = truncateBytes(text, maxAPIToolResultBytes)
		content["truncated"] = true
		content["body_bytes"] = len(text)
	case structured:
		content["body"] = parseJSONBody(text)
	case text != "":
		content["body"] = text
	}
	return content, nil
}

// toolCallConfig turns tool call arguments into the openapi node config.
// Arguments are grouped as in openapi.ToolParameters. Models sometimes flatten
// them, so a declared parameter at the top level is moved to its group, and
// other top-level values become fields of the body when no body is given.
func toolCallConfig(op *models.APIOperation, args map[string]any) (map[string]any, error) {
	config := make(map[string]any)
	for _, key := range toolArgGroups {
		value, ok := args[key]
		if !ok || value == nil {
			continue
		}
		group, err := argumentGroup(key, value)
		if err != nil {
			return nil, err
		}
		config[key] = group
	}
	body, hasBody := args[openapi.ToolArgBody]
	if hasBody {
		config[openapi.ToolArgBody] = body
	}

	var loose []string
	for key := range args {
		if !isToolArgKey(key) {
			loose = append(loose, key)
		}
	}
	sort.Strings(loose)
	var unknown []string
	bodyFields := make(map[string]any)
	for _, key := range loose {
		if param := declaredParam(op, key); param != nil {
			groupKey := openapi.ToolArgKey(param.In)
			group, _ := config[groupKey].(map[string]any)
			if group == nil {
				group = make(map[string]any)
				config[groupKey] = group
			}
			if _, exists := group[param.Name]; !exists {
				group[param.Name] = args[key]
			}
			continue
		}
		if op.RequestBody != nil && !hasBody {
			bodyFields[key] = args[key]
			continue
		}
		unknown = append(unknown, key)
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown arguments: %s; parameters go in %s, the request body in %q",
			strings.Join(unknown, ", "), strings.Join(toolArgGroups, ", "), openapi.ToolArgBody)
	}
	if len(bodyFields) > 0 {
		config[openapi.ToolArgBody] = bodyFields
	}
	return config, nil
}

func argumentGroup(key string, value any) (map[string]any, error) {
	switch v := value.(type) {
	case map[string]any:
		group := make(map[string]any, len(v))
		for name, item := range v {
			group[name] = item
		}
		return group, nil
	case string:
		if group := objectConfig(map[string]any{key: v}, key); group != nil {
			return group, nil
		}
		if strings.TrimSpace(v) == "" {
			return map[string]any{}, nil
		}
	}
	return nil, fmt.Errorf("argument %q must be an object", key)
}

func isToolArgKey(key string) bool {
	if key == openapi.ToolArgBody {
		return true
	}
	for _, group := range toolArgGroups {
		if key == group {
			return true
		}
	}
	return false
}

// declaredParam finds a parameter by name, preferring path, query, header and
// cookie parameters in that order. Header names are compared case-insensitively.
func declaredParam(op *models.APIOperation, name string) *models.APIParameter {
	for _, in := range []string{"path", "query", "header", "cookie"} {
		for _, p := range op.Parameters {
			if p.In == in && (p.Name == name || (in == "header" && strings.EqualFold(p.Name, name))) {
				return p
			}
		}
	}
	return nil
}

// bodyText renders a response body as text; structured reports a JSON value
// that is not a plain string.
func bodyText(body any) (text string, structured bool) {
	switch v := body.(type) {
	case nil:
		return "", false
	case string:
		return v, false
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprint(body), false
	}
	return string(data), true
}

// parseJSONBody decodes a body rendered by bodyText. Masking a secret can
// break the JSON, and then the text is returned as is.
func parseJSONBody(text string) any {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return text
	}
	return value
}

// truncateBytes cuts s to at most limit bytes at a rune boundary and marks the cut.
func truncateBytes(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…[truncated]"
}

// toolName turns an operation ID into a function name that LLM providers
// accept: letters, digits, "_" and "-", starting with a letter or "_", at most
// maxToolNameLength characters.
func toolName(operationID string) string {
	name := toolNameInvalidChars.ReplaceAllString(operationID, "_")
	if name == "" || (name[0] != '_' && !isASCIILetter(name[0])) {
		name = "op_" + name
	}
	if len(name) > maxToolNameLength {
		name = name[:maxToolNameLength]
	}
	return name
}

func isASCIILetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

// uniqueToolName returns name, or name with a numeric suffix when taken has it.
func uniqueToolName(name string, taken map[string]bool) string {
	if !taken[name] {
		return name
	}
	for n := 2; ; n++ {
		suffix := "_" + strconv.Itoa(n)
		base := name
		if len(base)+len(suffix) > maxToolNameLength {
			base = base[:maxToolNameLength-len(suffix)]
		}
		if candidate := base + suffix; !taken[candidate] {
			return candidate
		}
	}
}

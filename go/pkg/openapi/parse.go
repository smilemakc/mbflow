// Package openapi turns OpenAPI 3.x and Swagger 2.0 documents into API client
// operations and builds HTTP requests for those operations.
package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oasdiff/yaml"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

const (
	// MaxSpecSize bounds the size of a specification document in bytes.
	MaxSpecSize = 32 << 20
	// MaxOperations bounds the number of operations in one specification.
	MaxOperations = 5000

	maxWarningLength = 500
)

// Warning codes reported in Spec.Warnings.
const (
	WarningNoServers       = "no_servers"
	WarningRelativeServer  = "relative_server"
	WarningGeneratedIDs    = "generated_operation_ids"
	WarningDuplicateIDs    = "duplicate_operation_ids"
	WarningValidation      = "validation"
	WarningUnsupportedAuth = "unsupported_security"
)

const (
	defaultUntitledSpecTitle = "Untitled API"
	swaggerSpecVersion       = "2.0"
)

// Spec is the normalized content of an OpenAPI or Swagger document.
type Spec struct {
	Title           string
	Description     string
	APIVersion      string
	SpecVersion     string
	Servers         []models.APIServer
	SecuritySchemes []*models.APISecurityScheme
	SuggestedAuth   models.APIClientAuth
	Operations      []*models.APIOperation
	Schemas         map[string]map[string]any
	Warnings        []models.APISpecWarning
}

// DefaultBaseURL returns the first absolute server URL, or "" when the
// specification declares none.
func (s *Spec) DefaultBaseURL() string {
	for _, server := range s.Servers {
		if isAbsoluteHTTPURL(server.URL) {
			return server.URL
		}
	}
	return ""
}

// Parse reads an OpenAPI 3.x or Swagger 2.0 document in JSON or YAML.
// External $ref values are rejected: the document must be self-contained.
// Errors wrap models.ErrInvalidAPISpec.
func Parse(ctx context.Context, data []byte) (*Spec, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, invalidSpec("document is empty")
	}
	if len(data) > MaxSpecSize {
		return nil, invalidSpec(fmt.Sprintf("document exceeds %d MB", MaxSpecSize>>20))
	}

	version, err := detectVersion(data)
	if err != nil {
		return nil, invalidSpec("document is neither valid JSON nor YAML: " + err.Error())
	}
	if version.numeric {
		if data, err = withStringVersion(data, version); err != nil {
			return nil, invalidSpec(err.Error())
		}
	}

	loader := openapi3.NewLoader()
	loader.Context = ctx
	loader.IsExternalRefsAllowed = false

	var doc *openapi3.T
	var specVersion string
	switch {
	case strings.HasPrefix(version.openapi, "3."):
		doc, err = loader.LoadFromData(data)
		if err != nil {
			return nil, invalidSpec(err.Error())
		}
		specVersion = version.openapi
	case version.swagger == "2" || version.swagger == swaggerSpecVersion:
		var doc2 openapi2.T
		if _, err := yaml.Unmarshal(data, &doc2, yaml.DecodeOpts{DisableTimestamps: true}); err != nil {
			return nil, invalidSpec(err.Error())
		}
		doc, err = openapi2conv.ToV3WithLoader(&doc2, loader, nil)
		if err != nil {
			return nil, invalidSpec(err.Error())
		}
		if doc2.Host == "" && doc2.BasePath != "" && doc2.BasePath != "/" {
			// ToV3 drops basePath when host is absent; keep it as a relative server.
			doc.Servers = append(doc.Servers, &openapi3.Server{URL: doc2.BasePath})
		}
		specVersion = swaggerSpecVersion
	default:
		return nil, invalidSpec(`expected "openapi: 3.x" or "swagger: 2.0" at the document root`)
	}

	b := newSpecBuilder(doc, specVersion)
	if err := b.build(ctx); err != nil {
		return nil, err
	}
	return b.spec, nil
}

func invalidSpec(msg string) error {
	return fmt.Errorf("%w: %s", models.ErrInvalidAPISpec, msg)
}

type specVersionInfo struct {
	openapi string
	swagger string
	// numeric is set for unquoted YAML versions such as "swagger: 2.0",
	// which decode as numbers and fail typed decoding.
	numeric bool
}

func detectVersion(data []byte) (specVersionInfo, error) {
	var probe struct {
		OpenAPI any `json:"openapi"`
		Swagger any `json:"swagger"`
	}
	trimmed := bytes.TrimSpace(data)
	var err error
	if trimmed[0] == '{' {
		err = json.Unmarshal(trimmed, &probe)
	} else {
		_, err = yaml.Unmarshal(trimmed, &probe, yaml.DecodeOpts{DisableTimestamps: true})
	}
	if err != nil {
		return specVersionInfo{}, err
	}
	info := specVersionInfo{}
	info.openapi, info.numeric = versionString(probe.OpenAPI)
	var swaggerNumeric bool
	info.swagger, swaggerNumeric = versionString(probe.Swagger)
	info.numeric = info.numeric || swaggerNumeric
	if info.swagger == "2" {
		info.swagger = swaggerSpecVersion
	}
	return info, nil
}

func versionString(v any) (version string, numeric bool) {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t), false
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	default:
		return "", false
	}
}

// withStringVersion re-encodes the document as JSON with the version fields as strings.
func withStringVersion(data []byte, version specVersionInfo) ([]byte, error) {
	var generic map[string]any
	if _, err := yaml.Unmarshal(data, &generic, yaml.DecodeOpts{DisableTimestamps: true}); err != nil {
		return nil, err
	}
	if version.openapi != "" {
		generic["openapi"] = version.openapi
	}
	if version.swagger != "" {
		generic["swagger"] = version.swagger
	}
	return json.Marshal(generic)
}

type specBuilder struct {
	doc          *openapi3.T
	spec         *Spec
	schemas      *schemaConverter
	generatedIDs int
	duplicateIDs int
}

func newSpecBuilder(doc *openapi3.T, specVersion string) *specBuilder {
	return &specBuilder{
		doc:     doc,
		spec:    &Spec{SpecVersion: specVersion},
		schemas: newSchemaConverter(doc),
	}
}

func (b *specBuilder) build(ctx context.Context) error {
	if info := b.doc.Info; info != nil {
		b.spec.Title = strings.TrimSpace(info.Title)
		b.spec.Description = strings.TrimSpace(info.Description)
		b.spec.APIVersion = strings.TrimSpace(info.Version)
	}
	if b.spec.Title == "" {
		b.spec.Title = defaultUntitledSpecTitle
	}

	b.spec.Servers = convertServers(b.doc.Servers)
	b.spec.SecuritySchemes = b.convertSecuritySchemes()

	if err := b.collectOperations(); err != nil {
		return err
	}
	if len(b.spec.Operations) == 0 {
		return invalidSpec("specification declares no operations")
	}

	b.spec.Schemas = b.schemas.components()
	b.spec.SuggestedAuth = b.suggestAuth()
	b.addServerWarnings()

	if b.generatedIDs > 0 {
		b.warn(WarningGeneratedIDs, fmt.Sprintf(
			"%d operation(s) have no operationId; their IDs are generated from method and path", b.generatedIDs))
	}
	if b.duplicateIDs > 0 {
		b.warn(WarningDuplicateIDs, fmt.Sprintf(
			"%d operation(s) share an operationId; their IDs are generated from method and path", b.duplicateIDs))
	}

	if err := b.doc.Validate(ctx,
		openapi3.DisableExamplesValidation(),
		openapi3.DisableSchemaDefaultsValidation(),
		openapi3.DisableSchemaPatternValidation(),
	); err != nil {
		b.warn(WarningValidation, "specification does not fully conform to the standard: "+err.Error())
	}
	return nil
}

func (b *specBuilder) warn(code, message string) {
	if len(message) > maxWarningLength {
		message = message[:maxWarningLength] + "…"
	}
	b.spec.Warnings = append(b.spec.Warnings, models.APISpecWarning{Code: code, Message: message})
}

func (b *specBuilder) addServerWarnings() {
	if len(b.spec.Servers) == 0 {
		b.warn(WarningNoServers, "specification declares no servers; set the base URL of the API client")
		return
	}
	if b.spec.DefaultBaseURL() == "" {
		b.warn(WarningRelativeServer, fmt.Sprintf(
			"server URL %q is relative; set the absolute base URL of the API client, e.g. https://api.example.com%s",
			b.spec.Servers[0].URL, strings.TrimSuffix(b.spec.Servers[0].URL, "/")))
	}
}

var methodOrder = map[string]int{
	http.MethodGet:     0,
	http.MethodPost:    1,
	http.MethodPut:     2,
	http.MethodPatch:   3,
	http.MethodDelete:  4,
	http.MethodHead:    5,
	http.MethodOptions: 6,
	http.MethodTrace:   7,
}

type pendingOperation struct {
	path   string
	method string
	item   *openapi3.PathItem
	op     *openapi3.Operation
}

func (b *specBuilder) collectOperations() error {
	if b.doc.Paths == nil {
		return nil
	}
	paths := b.doc.Paths.Map()
	var pending []pendingOperation
	for _, path := range sortedKeys(paths) {
		item := paths[path]
		if item == nil {
			continue
		}
		operations := item.Operations()
		methods := make([]string, 0, len(operations))
		for m := range operations {
			methods = append(methods, m)
		}
		sort.Slice(methods, func(i, j int) bool {
			oi, iKnown := methodOrder[strings.ToUpper(methods[i])]
			oj, jKnown := methodOrder[strings.ToUpper(methods[j])]
			if iKnown != jKnown {
				return iKnown
			}
			if oi != oj {
				return oi < oj
			}
			return methods[i] < methods[j]
		})

		for _, method := range methods {
			if op := operations[method]; op != nil {
				pending = append(pending, pendingOperation{path: path, method: strings.ToUpper(method), item: item, op: op})
			}
		}
	}
	if len(pending) > MaxOperations {
		return invalidSpec(fmt.Sprintf("specification has %d operations, at most %d are supported", len(pending), MaxOperations))
	}

	keys := b.assignOperationKeys(pending)
	for i, p := range pending {
		b.spec.Operations = append(b.spec.Operations, b.convertOperation(keys[i], p))
	}
	return nil
}

// assignOperationKeys uses unique operationIds as keys. Operations without an
// operationId, or sharing one, get keys derived from method and path, so a key
// does not depend on which other operations the document contains.
func (b *specBuilder) assignOperationKeys(pending []pendingOperation) []string {
	counts := make(map[string]int)
	for _, p := range pending {
		if id := strings.TrimSpace(p.op.OperationID); id != "" {
			counts[id]++
		}
	}

	keys := make([]string, len(pending))
	used := make(map[string]bool)
	for i, p := range pending {
		if id := strings.TrimSpace(p.op.OperationID); id != "" && counts[id] == 1 {
			keys[i] = id
			used[id] = true
		}
	}
	for i, p := range pending {
		if keys[i] != "" {
			continue
		}
		if strings.TrimSpace(p.op.OperationID) == "" {
			b.generatedIDs++
		} else {
			b.duplicateIDs++
		}
		base := generateOperationKey(p.method, p.path)
		key := base
		for n := 2; used[key]; n++ {
			key = base + "_" + strconv.Itoa(n)
		}
		used[key] = true
		keys[i] = key
	}
	return keys
}

func (b *specBuilder) convertOperation(key string, p pendingOperation) *models.APIOperation {
	op, item := p.op, p.item
	result := &models.APIOperation{
		ID:          key,
		OperationID: strings.TrimSpace(op.OperationID),
		Method:      p.method,
		Path:        p.path,
		Summary:     strings.TrimSpace(op.Summary),
		Description: strings.TrimSpace(op.Description),
		Tags:        op.Tags,
		Deprecated:  op.Deprecated,
	}
	if result.Summary == "" {
		result.Summary = strings.TrimSpace(item.Summary)
	}
	result.Parameters = b.convertParameters(p.path, item.Parameters, op.Parameters)

	if op.RequestBody != nil && op.RequestBody.Value != nil {
		result.RequestBody = b.convertRequestBody(op.RequestBody.Value)
	}
	if op.Responses != nil {
		result.Responses = b.convertResponses(op.Responses)
	}

	if op.Servers != nil {
		result.Servers = convertServers(*op.Servers)
	} else {
		result.Servers = convertServers(item.Servers)
	}

	security := b.doc.Security
	if op.Security != nil {
		security = *op.Security
		result.NoAuth = len(security) == 0
	}
	result.Security = convertSecurityRequirements(security)
	return result
}

var nonIdentifierChars = regexp.MustCompile(`[^A-Za-z0-9]+`)

// generateOperationKey builds a stable key such as "get_pets_petId" from method and path.
func generateOperationKey(method, path string) string {
	segment := strings.Trim(nonIdentifierChars.ReplaceAllString(path, "_"), "_")
	if segment == "" {
		segment = "root"
	}
	return strings.ToLower(method) + "_" + segment
}

type paramKey struct {
	name string
	in   string
}

// ignoredHeaderParams are header parameters the OpenAPI specification says to ignore.
var ignoredHeaderParams = map[string]bool{
	"accept":        true,
	"content-type":  true,
	"authorization": true,
}

var pathTemplateParam = regexp.MustCompile(`\{([^{}]+)\}`)

func (b *specBuilder) convertParameters(path string, pathLevel, opLevel openapi3.Parameters) []*models.APIParameter {
	var order []paramKey
	byKey := make(map[paramKey]*models.APIParameter)

	add := func(ref *openapi3.ParameterRef) {
		if ref == nil || ref.Value == nil {
			return
		}
		p := ref.Value
		if p.In == openapi3.ParameterInHeader && ignoredHeaderParams[strings.ToLower(p.Name)] {
			return
		}
		key := paramKey{name: p.Name, in: p.In}
		if _, exists := byKey[key]; !exists {
			order = append(order, key)
		}
		byKey[key] = b.convertParameter(p)
	}
	for _, ref := range pathLevel {
		add(ref)
	}
	for _, ref := range opLevel {
		add(ref)
	}

	// Path template variables missing from the declared parameters still
	// have to be filled in, so they are added as required strings.
	for _, match := range pathTemplateParam.FindAllStringSubmatch(path, -1) {
		key := paramKey{name: match[1], in: openapi3.ParameterInPath}
		if _, exists := byKey[key]; exists {
			continue
		}
		order = append(order, key)
		byKey[key] = &models.APIParameter{
			Name:     match[1],
			In:       openapi3.ParameterInPath,
			Required: true,
			Schema:   map[string]any{"type": "string"},
		}
	}

	if len(order) == 0 {
		return nil
	}
	result := make([]*models.APIParameter, 0, len(order))
	for _, key := range order {
		result = append(result, byKey[key])
	}
	return result
}

func (b *specBuilder) convertParameter(p *openapi3.Parameter) *models.APIParameter {
	param := &models.APIParameter{
		Name:        p.Name,
		In:          p.In,
		Description: strings.TrimSpace(p.Description),
		Required:    p.Required || p.In == openapi3.ParameterInPath,
		Deprecated:  p.Deprecated,
		Style:       p.Style,
		Explode:     p.Explode,
		Example:     p.Example,
	}
	if p.Schema != nil {
		param.Schema = b.schemas.convert(p.Schema)
	} else if len(p.Content) > 0 {
		contentTypes := sortedContentTypes(p.Content)
		param.ContentType = contentTypes[0]
		if mt := p.Content[param.ContentType]; mt != nil && mt.Schema != nil {
			param.Schema = b.schemas.convert(mt.Schema)
		}
	}
	if param.Example == nil {
		param.Example = firstExample(p.Examples)
	}
	return param
}

func (b *specBuilder) convertRequestBody(body *openapi3.RequestBody) *models.APIRequestBody {
	result := &models.APIRequestBody{
		Description: strings.TrimSpace(body.Description),
		Required:    body.Required,
	}
	result.Content = b.convertContent(body.Content)
	if len(result.Content) == 0 {
		result.Content = []*models.APIMediaType{{ContentType: "application/json"}}
	}
	return result
}

func (b *specBuilder) convertContent(content openapi3.Content) []*models.APIMediaType {
	if len(content) == 0 {
		return nil
	}
	result := make([]*models.APIMediaType, 0, len(content))
	for _, contentType := range sortedContentTypes(content) {
		media := &models.APIMediaType{ContentType: contentType}
		if mt := content[contentType]; mt != nil {
			if mt.Schema != nil {
				media.Schema = b.schemas.convert(mt.Schema)
			}
			media.Example = mt.Example
			if media.Example == nil {
				media.Example = firstExample(mt.Examples)
			}
		}
		result = append(result, media)
	}
	return result
}

func (b *specBuilder) convertResponses(responses *openapi3.Responses) []*models.APIResponse {
	m := responses.Map()
	if len(m) == 0 {
		return nil
	}
	statuses := make([]string, 0, len(m))
	for status := range m {
		statuses = append(statuses, status)
	}
	sort.Slice(statuses, func(i, j int) bool { return statusRank(statuses[i]) < statusRank(statuses[j]) })

	result := make([]*models.APIResponse, 0, len(statuses))
	for _, status := range statuses {
		ref := m[status]
		if ref == nil || ref.Value == nil {
			continue
		}
		resp := &models.APIResponse{Status: status}
		if ref.Value.Description != nil {
			resp.Description = strings.TrimSpace(*ref.Value.Description)
		}
		resp.Content = b.convertContent(ref.Value.Content)
		result = append(result, resp)
	}
	return result
}

// statusRank orders exact codes first, then ranges like "2XX", then "default".
func statusRank(status string) int {
	if code, err := strconv.Atoi(status); err == nil {
		return code
	}
	upper := strings.ToUpper(status)
	if len(upper) == 3 && strings.HasSuffix(upper, "XX") && upper[0] >= '1' && upper[0] <= '5' {
		return 1000 + int(upper[0]-'0')*100
	}
	return 10000
}

func (b *specBuilder) convertSecuritySchemes() []*models.APISecurityScheme {
	if b.doc.Components == nil || len(b.doc.Components.SecuritySchemes) == 0 {
		return nil
	}
	names := make([]string, 0, len(b.doc.Components.SecuritySchemes))
	for name := range b.doc.Components.SecuritySchemes {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]*models.APISecurityScheme, 0, len(names))
	for _, name := range names {
		ref := b.doc.Components.SecuritySchemes[name]
		if ref == nil || ref.Value == nil {
			continue
		}
		s := ref.Value
		scheme := &models.APISecurityScheme{
			Name:         name,
			Type:         s.Type,
			Scheme:       strings.ToLower(s.Scheme),
			BearerFormat: s.BearerFormat,
			In:           s.In,
			ParamName:    s.Name,
			Description:  strings.TrimSpace(s.Description),
		}
		if s.Flows != nil && s.Flows.ClientCredentials != nil {
			scheme.TokenURL = s.Flows.ClientCredentials.TokenURL
			scheme.Scopes = sortedKeys(s.Flows.ClientCredentials.Scopes)
		}
		result = append(result, scheme)
	}
	return result
}

// suggestAuth picks the scheme required at the document level, then the one
// used by most operations, then the first declared one.
func (b *specBuilder) suggestAuth() models.APIClientAuth {
	byName := make(map[string]*models.APISecurityScheme, len(b.spec.SecuritySchemes))
	for _, s := range b.spec.SecuritySchemes {
		byName[s.Name] = s
	}

	var candidates []string
	seen := make(map[string]bool)
	addCandidate := func(name string) {
		if !seen[name] {
			seen[name] = true
			candidates = append(candidates, name)
		}
	}
	for _, requirement := range convertSecurityRequirements(b.doc.Security) {
		for _, name := range requirement {
			addCandidate(name)
		}
	}
	usage := make(map[string]int)
	for _, op := range b.spec.Operations {
		for _, requirement := range op.Security {
			for _, name := range requirement {
				usage[name]++
			}
		}
	}
	used := sortedKeys(usage)
	sort.SliceStable(used, func(i, j int) bool { return usage[used[i]] > usage[used[j]] })
	for _, name := range used {
		addCandidate(name)
	}
	for _, s := range b.spec.SecuritySchemes {
		addCandidate(s.Name)
	}

	var unsupported []string
	for _, name := range candidates {
		scheme, ok := byName[name]
		if !ok {
			continue
		}
		if auth, ok := authFromScheme(scheme); ok {
			return auth
		}
		unsupported = append(unsupported, fmt.Sprintf("%s (%s)", scheme.Name, scheme.Type))
	}
	if len(unsupported) > 0 {
		b.warn(WarningUnsupportedAuth, fmt.Sprintf(
			"security schemes %s are not supported; pass authentication headers in the node settings",
			strings.Join(unsupported, ", ")))
	}
	return models.APIClientAuth{Type: models.APIClientAuthNone}
}

func authFromScheme(s *models.APISecurityScheme) (models.APIClientAuth, bool) {
	switch strings.ToLower(s.Type) {
	case "apikey":
		if s.In != models.APIKeyInHeader && s.In != models.APIKeyInQuery && s.In != models.APIKeyInCookie {
			return models.APIClientAuth{}, false
		}
		return models.APIClientAuth{
			Type:       models.APIClientAuthAPIKey,
			In:         s.In,
			Name:       s.ParamName,
			SchemeName: s.Name,
		}, s.ParamName != ""
	case "http":
		switch s.Scheme {
		case "basic":
			return models.APIClientAuth{Type: models.APIClientAuthBasic, SchemeName: s.Name}, true
		case "bearer":
			return models.APIClientAuth{Type: models.APIClientAuthBearer, SchemeName: s.Name}, true
		}
	case "oauth2":
		if s.TokenURL != "" {
			return models.APIClientAuth{
				Type:       models.APIClientAuthOAuth2ClientCredentials,
				SchemeName: s.Name,
				TokenURL:   s.TokenURL,
				Scopes:     s.Scopes,
			}, true
		}
		// Other OAuth2 flows need a user-obtained token, which is sent as a bearer token.
		return models.APIClientAuth{Type: models.APIClientAuthBearer, SchemeName: s.Name}, true
	case "openidconnect":
		return models.APIClientAuth{Type: models.APIClientAuthBearer, SchemeName: s.Name}, true
	}
	return models.APIClientAuth{}, false
}

func convertSecurityRequirements(requirements openapi3.SecurityRequirements) [][]string {
	if len(requirements) == 0 {
		return nil
	}
	result := make([][]string, 0, len(requirements))
	for _, requirement := range requirements {
		result = append(result, sortedKeys(requirement))
	}
	return result
}

func convertServers(servers openapi3.Servers) []models.APIServer {
	var result []models.APIServer
	for _, s := range servers {
		if s == nil || strings.TrimSpace(s.URL) == "" {
			continue
		}
		u := strings.TrimSpace(s.URL)
		for name, variable := range s.Variables {
			if variable != nil {
				u = strings.ReplaceAll(u, "{"+name+"}", variable.Default)
			}
		}
		if u != "/" {
			u = strings.TrimRight(u, "/")
		}
		result = append(result, models.APIServer{URL: u, Description: strings.TrimSpace(s.Description)})
	}
	return result
}

func isAbsoluteHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// contentTypeRank orders media types by how well the request builder supports them.
func contentTypeRank(contentType string) int {
	mediaType := baseMediaType(contentType)
	switch {
	case mediaType == "application/json":
		return 0
	case isJSONMediaType(mediaType):
		return 1
	case mediaType == "application/x-www-form-urlencoded":
		return 2
	case mediaType == "multipart/form-data":
		return 3
	case strings.HasPrefix(mediaType, "text/"):
		return 4
	default:
		return 5
	}
}

func sortedContentTypes(content openapi3.Content) []string {
	types := make([]string, 0, len(content))
	for ct := range content {
		types = append(types, ct)
	}
	sort.Slice(types, func(i, j int) bool {
		ri, rj := contentTypeRank(types[i]), contentTypeRank(types[j])
		if ri != rj {
			return ri < rj
		}
		return types[i] < types[j]
	})
	return types
}

func firstExample(examples openapi3.Examples) any {
	for _, name := range sortedKeys(examples) {
		if ref := examples[name]; ref != nil && ref.Value != nil && ref.Value.Value != nil {
			return ref.Value.Value
		}
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

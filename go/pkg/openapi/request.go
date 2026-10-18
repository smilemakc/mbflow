package openapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ErrBaseURLRequired is returned when neither the node, the operation nor the
// API client provides an absolute base URL.
var ErrBaseURLRequired = errors.New("base URL is not configured")

// RequestInput holds the values a workflow node passes to an operation.
// Maps are keyed by parameter name; keys that the operation does not declare
// are sent as plain query parameters, headers or cookies.
type RequestInput struct {
	BaseURL     string
	PathParams  map[string]any
	QueryParams map[string]any
	Headers     map[string]any
	Cookies     map[string]any
	Body        any
	ContentType string
}

// Request is an HTTP request built for an operation, before authentication.
type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   []byte
}

// BuildRequest builds the HTTP request of an operation of the client.
func BuildRequest(client *models.APIClient, op *models.APIOperation, in RequestInput) (*Request, error) {
	base, err := resolveBaseURL(client, op, in.BaseURL)
	if err != nil {
		return nil, err
	}
	path, err := expandPath(op, in.PathParams, client.Schemas)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, fmt.Errorf("invalid request URL: %w", err)
	}

	// The executor adds authentication after the request is built, so
	// parameters it fills in are not required from the caller.
	setByAuth := func(p *models.APIParameter) bool { return isAuthParam(client, op, p) }

	query := u.Query()
	if err := addQueryParams(query, op, in.QueryParams, client.Schemas, setByAuth); err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()

	header := make(http.Header)
	for name, value := range client.DefaultHeaders {
		header.Set(name, value)
	}
	if err := addHeaderParams(header, op, in.Headers, client.Schemas, setByAuth); err != nil {
		return nil, err
	}
	cookies, err := cookieParams(op, in.Cookies, client.Schemas, setByAuth)
	if err != nil {
		return nil, err
	}
	if len(cookies) > 0 {
		header.Set("Cookie", strings.Join(cookies, "; "))
	}

	body, contentType, err := encodeBody(op, in.Body, in.ContentType, client.Schemas)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	if header.Get("Accept") == "" {
		if accept := acceptHeader(op); accept != "" {
			header.Set("Accept", accept)
		}
	}

	return &Request{Method: op.Method, URL: u, Header: header, Body: body}, nil
}

// resolveBaseURL prefers the node override, then an absolute server declared
// by the operation, then the client base URL.
func resolveBaseURL(client *models.APIClient, op *models.APIOperation, override string) (string, error) {
	base := strings.TrimSpace(override)
	if base == "" {
		for _, server := range op.Servers {
			if isAbsoluteHTTPURL(server.URL) {
				base = server.URL
				break
			}
		}
	}
	if base == "" {
		base = client.BaseURL
	}
	if base == "" {
		return "", fmt.Errorf("%w for API client %q", ErrBaseURLRequired, client.Name)
	}
	if !isAbsoluteHTTPURL(base) {
		return "", fmt.Errorf("base URL %q must be an absolute http or https URL", base)
	}
	return strings.TrimRight(base, "/"), nil
}

func findParam(op *models.APIOperation, name, in string) *models.APIParameter {
	for _, p := range op.Parameters {
		if p.Name == name && p.In == in {
			return p
		}
	}
	return nil
}

func expandPath(op *models.APIOperation, values map[string]any, schemas map[string]map[string]any) (string, error) {
	var missing []string
	path := pathTemplateParam.ReplaceAllStringFunc(op.Path, func(match string) string {
		name := match[1 : len(match)-1]
		value, ok := values[name]
		if !ok || isEmptyValue(value) {
			missing = append(missing, name)
			return match
		}
		param := findParam(op, name, "path")
		if param == nil {
			param = &models.APIParameter{Name: name, In: "path"}
		}
		return serializePathParam(param, normalizeParamValue(value, param.Schema, schemas))
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("missing required path parameters: %s", strings.Join(missing, ", "))
	}
	return path, nil
}

func serializePathParam(p *models.APIParameter, value any) string {
	style := p.Style
	if style == "" {
		style = "simple"
	}
	explode := p.Explode != nil && *p.Explode

	var prefix, itemSep, pairSep string
	switch style {
	case "label":
		prefix, itemSep, pairSep = ".", ",", ","
		if explode {
			itemSep, pairSep = ".", "."
		}
	case "matrix":
		prefix, itemSep, pairSep = ";"+url.PathEscape(p.Name)+"=", ",", ","
	default:
		itemSep, pairSep = ",", ","
	}

	switch v := value.(type) {
	case []any:
		items := escapeAll(stringSlice(v), url.PathEscape)
		if style == "matrix" && explode {
			return ";" + url.PathEscape(p.Name) + "=" + strings.Join(items, ";"+url.PathEscape(p.Name)+"=")
		}
		return prefix + strings.Join(items, itemSep)
	case map[string]any:
		keys := sortedKeys(v)
		parts := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			key, val := url.PathEscape(k), url.PathEscape(stringValue(v[k]))
			if explode {
				parts = append(parts, key+"="+val)
			} else {
				parts = append(parts, key, val)
			}
		}
		if style == "matrix" && explode {
			return ";" + strings.Join(parts, ";")
		}
		return prefix + strings.Join(parts, pairSep)
	default:
		return prefix + url.PathEscape(stringValue(v))
	}
}

func addQueryParams(query url.Values, op *models.APIOperation, values map[string]any, schemas map[string]map[string]any, setByAuth func(*models.APIParameter) bool) error {
	declared := make(map[string]bool)
	var missing []string
	for _, p := range op.Parameters {
		if p.In != "query" {
			continue
		}
		declared[p.Name] = true
		value, ok := values[p.Name]
		if !ok || isEmptyValue(value) {
			if p.Required && !setByAuth(p) {
				missing = append(missing, p.Name)
			}
			continue
		}
		value = normalizeParamValue(value, p.Schema, schemas)
		if p.ContentType != "" {
			query.Add(p.Name, jsonString(value))
			continue
		}

		style := p.Style
		if style == "" {
			style = "form"
		}
		explode := style == "form"
		if p.Explode != nil {
			explode = *p.Explode
		}

		switch v := value.(type) {
		case []any:
			items := stringSlice(v)
			switch {
			case style == "spaceDelimited":
				query.Add(p.Name, strings.Join(items, " "))
			case style == "pipeDelimited":
				query.Add(p.Name, strings.Join(items, "|"))
			case explode:
				for _, item := range items {
					query.Add(p.Name, item)
				}
			default:
				query.Add(p.Name, strings.Join(items, ","))
			}
		case map[string]any:
			keys := sortedKeys(v)
			switch {
			case style == "deepObject":
				for _, k := range keys {
					query.Add(p.Name+"["+k+"]", stringValue(v[k]))
				}
			case explode:
				for _, k := range keys {
					query.Add(k, stringValue(v[k]))
				}
			default:
				parts := make([]string, 0, len(keys)*2)
				for _, k := range keys {
					parts = append(parts, k, stringValue(v[k]))
				}
				query.Add(p.Name, strings.Join(parts, ","))
			}
		default:
			query.Add(p.Name, stringValue(v))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required query parameters: %s", strings.Join(missing, ", "))
	}

	for _, name := range sortedKeys(values) {
		if declared[name] || isEmptyValue(values[name]) {
			continue
		}
		if list, ok := values[name].([]any); ok {
			for _, item := range stringSlice(list) {
				query.Add(name, item)
			}
			continue
		}
		query.Add(name, stringValue(values[name]))
	}
	return nil
}

func addHeaderParams(header http.Header, op *models.APIOperation, values map[string]any, schemas map[string]map[string]any, setByAuth func(*models.APIParameter) bool) error {
	declared := make(map[string]bool)
	var missing []string
	for _, p := range op.Parameters {
		if p.In != "header" {
			continue
		}
		declared[strings.ToLower(p.Name)] = true
		value, ok := lookupFold(values, p.Name)
		if !ok || isEmptyValue(value) {
			if p.Required && !setByAuth(p) {
				missing = append(missing, p.Name)
			}
			continue
		}
		explode := p.Explode != nil && *p.Explode
		header.Set(p.Name, serializeSimple(normalizeParamValue(value, p.Schema, schemas), explode))
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required header parameters: %s", strings.Join(missing, ", "))
	}
	for _, name := range sortedKeys(values) {
		if declared[strings.ToLower(name)] || isEmptyValue(values[name]) || strings.TrimSpace(name) == "" {
			continue
		}
		header.Set(name, serializeSimple(values[name], false))
	}
	return nil
}

func cookieParams(op *models.APIOperation, values map[string]any, schemas map[string]map[string]any, setByAuth func(*models.APIParameter) bool) ([]string, error) {
	declared := make(map[string]bool)
	var cookies, missing []string
	for _, p := range op.Parameters {
		if p.In != "cookie" {
			continue
		}
		declared[p.Name] = true
		value, ok := values[p.Name]
		if !ok || isEmptyValue(value) {
			if p.Required && !setByAuth(p) {
				missing = append(missing, p.Name)
			}
			continue
		}
		cookies = append(cookies, p.Name+"="+url.QueryEscape(serializeSimple(normalizeParamValue(value, p.Schema, schemas), false)))
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required cookie parameters: %s", strings.Join(missing, ", "))
	}
	for _, name := range sortedKeys(values) {
		if declared[name] || isEmptyValue(values[name]) {
			continue
		}
		cookies = append(cookies, name+"="+url.QueryEscape(serializeSimple(values[name], false)))
	}
	return cookies, nil
}

// serializeSimple applies the "simple" style used by headers.
func serializeSimple(value any, explode bool) string {
	switch v := value.(type) {
	case []any:
		return strings.Join(stringSlice(v), ",")
	case map[string]any:
		keys := sortedKeys(v)
		parts := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			if explode {
				parts = append(parts, k+"="+stringValue(v[k]))
			} else {
				parts = append(parts, k, stringValue(v[k]))
			}
		}
		return strings.Join(parts, ",")
	default:
		return stringValue(v)
	}
}

// normalizeParamValue turns a templated string into the array or object its
// schema declares: JSON text is decoded, other text is split on commas for arrays.
func normalizeParamValue(value any, schema map[string]any, schemas map[string]map[string]any) any {
	s, ok := value.(string)
	if !ok {
		return value
	}
	resolved := flattenAllOf(ResolveSchema(schemas, schema), schemas, 0)
	switch schemaType(resolved) {
	case "array":
		trimmed := strings.TrimSpace(s)
		if parsed, ok := parseJSON(trimmed); ok {
			if list, ok := parsed.([]any); ok {
				return list
			}
		}
		parts := strings.Split(trimmed, ",")
		list := make([]any, 0, len(parts))
		for _, part := range parts {
			if part = strings.TrimSpace(part); part != "" {
				list = append(list, part)
			}
		}
		return list
	case "object":
		if parsed, ok := parseJSON(strings.TrimSpace(s)); ok {
			if m, ok := parsed.(map[string]any); ok {
				return m
			}
		}
	}
	return value
}

func encodeBody(op *models.APIOperation, body any, contentTypeOverride string, schemas map[string]map[string]any) ([]byte, string, error) {
	rb := op.RequestBody
	if isEmptyBody(body, rb) {
		if rb != nil && rb.Required {
			return nil, "", errors.New("request body is required")
		}
		return nil, "", nil
	}

	contentType, schema := selectContentType(rb, contentTypeOverride)
	mediaType := baseMediaType(contentType)

	switch {
	case isJSONMediaType(mediaType) || strings.Contains(mediaType, "*"):
		data, err := encodeJSON(body, schema, schemas)
		if err != nil {
			return nil, "", err
		}
		if strings.Contains(contentType, "*") {
			contentType = "application/json"
		}
		return data, contentType, nil
	case mediaType == "application/x-www-form-urlencoded":
		data, err := encodeForm(body, schema, schemas)
		return data, contentType, err
	case mediaType == "multipart/form-data":
		return encodeMultipart(body, schema, schemas)
	default:
		data, err := encodeRaw(body, mediaType)
		return data, contentType, err
	}
}

// isEmptyBody treats nil, blank strings and, for optional bodies, empty
// objects as "no body": node forms keep an empty object when no field is set.
func isEmptyBody(body any, rb *models.APIRequestBody) bool {
	switch v := body.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case map[string]any:
		return len(v) == 0 && (rb == nil || !rb.Required)
	}
	return false
}

func selectContentType(rb *models.APIRequestBody, override string) (string, map[string]any) {
	override = strings.TrimSpace(override)
	if rb == nil || len(rb.Content) == 0 {
		if override != "" {
			return override, nil
		}
		return "application/json", nil
	}
	if override != "" {
		want := baseMediaType(override)
		for _, media := range rb.Content {
			if baseMediaType(media.ContentType) == want {
				return override, media.Schema
			}
		}
		return override, nil
	}
	return rb.Content[0].ContentType, rb.Content[0].Schema
}

func encodeJSON(body any, schema map[string]any, schemas map[string]map[string]any) ([]byte, error) {
	switch v := body.(type) {
	case []byte:
		return v, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if json.Valid([]byte(trimmed)) {
			return []byte(trimmed), nil
		}
		return json.Marshal(v)
	default:
		data, err := json.Marshal(Coerce(v, schema, schemas))
		if err != nil {
			return nil, fmt.Errorf("failed to encode JSON body: %w", err)
		}
		return data, nil
	}
}

func encodeForm(body any, schema map[string]any, schemas map[string]map[string]any) ([]byte, error) {
	switch v := body.(type) {
	case string:
		return []byte(v), nil
	case map[string]any:
		coerced, _ := Coerce(v, schema, schemas).(map[string]any)
		if coerced == nil {
			coerced = v
		}
		values := make(url.Values)
		for _, key := range sortedKeys(coerced) {
			switch item := coerced[key].(type) {
			case nil:
			case []any:
				for _, s := range stringSlice(item) {
					values.Add(key, s)
				}
			default:
				values.Add(key, stringValue(item))
			}
		}
		return []byte(values.Encode()), nil
	default:
		return nil, fmt.Errorf("form body must be an object, got %T", body)
	}
}

type filePart struct {
	filename    string
	contentType string
	data        []byte
}

var fileContentKeys = []string{"body_base64", "content_base64", "data_base64", "base64"}

// asFile recognizes file values in multipart bodies: objects carrying base64
// content (such as the binary output of an HTTP node, possibly serialized to
// JSON by a template) and plain strings for fields with format "binary".
func asFile(value any, schema map[string]any, fieldName string) (*filePart, bool) {
	if s, ok := value.(string); ok {
		if parsed, ok := parseJSON(strings.TrimSpace(s)); ok {
			if m, ok := parsed.(map[string]any); ok {
				return asFile(m, schema, fieldName)
			}
		}
		if schema != nil && schema["format"] == "binary" {
			return &filePart{filename: fieldName, contentType: "application/octet-stream", data: []byte(s)}, true
		}
		return nil, false
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	for _, key := range fileContentKeys {
		encoded, ok := m[key].(string)
		if !ok {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, false
		}
		file := &filePart{filename: fieldName, contentType: "application/octet-stream", data: data}
		for _, k := range []string{"filename", "file_name", "name"} {
			if s, ok := m[k].(string); ok && s != "" {
				file.filename = s
				break
			}
		}
		for _, k := range []string{"content_type", "mime_type"} {
			if s, ok := m[k].(string); ok && s != "" {
				file.contentType = s
				break
			}
		}
		return file, true
	}
	return nil, false
}

func encodeMultipart(body any, schema map[string]any, schemas map[string]map[string]any) ([]byte, string, error) {
	m, ok := body.(map[string]any)
	if !ok {
		if s, isString := body.(string); isString {
			if parsed, ok := parseJSON(strings.TrimSpace(s)); ok {
				m, _ = parsed.(map[string]any)
			}
		}
		if m == nil {
			return nil, "", fmt.Errorf("multipart body must be an object, got %T", body)
		}
	}

	resolved := flattenAllOf(ResolveSchema(schemas, schema), schemas, 0)
	props, _ := resolved["properties"].(map[string]any)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writeFile := func(field string, file *filePart) error {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf("form-data; name=%q; filename=%q", field, file.filename))
		h.Set("Content-Type", file.contentType)
		part, err := writer.CreatePart(h)
		if err != nil {
			return err
		}
		_, err = part.Write(file.data)
		return err
	}

	for _, key := range sortedKeys(m) {
		propSchema, _ := props[key].(map[string]any)
		propSchema = ResolveSchema(schemas, propSchema)
		value := m[key]
		if file, ok := asFile(value, propSchema, key); ok {
			if err := writeFile(key, file); err != nil {
				return nil, "", err
			}
			continue
		}
		switch v := value.(type) {
		case nil:
		case []any:
			items, _ := propSchema["items"].(map[string]any)
			items = ResolveSchema(schemas, items)
			for _, item := range v {
				if file, ok := asFile(item, items, key); ok {
					if err := writeFile(key, file); err != nil {
						return nil, "", err
					}
					continue
				}
				if err := writer.WriteField(key, stringValue(item)); err != nil {
					return nil, "", err
				}
			}
		default:
			if err := writer.WriteField(key, stringValue(Coerce(v, propSchema, schemas))); err != nil {
				return nil, "", err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}

func encodeRaw(body any, mediaType string) ([]byte, error) {
	switch v := body.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case map[string]any:
		if !strings.HasPrefix(mediaType, "text/") {
			if file, ok := asFile(v, nil, "body"); ok {
				return file.data, nil
			}
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode body: %w", err)
	}
	return data, nil
}

// acceptHeader lists the content types of the successful responses.
func acceptHeader(op *models.APIOperation) string {
	seen := make(map[string]bool)
	var types []string
	for _, resp := range op.Responses {
		if !strings.HasPrefix(resp.Status, "2") {
			continue
		}
		for _, media := range resp.Content {
			if !seen[media.ContentType] {
				seen[media.ContentType] = true
				types = append(types, media.ContentType)
			}
		}
	}
	return strings.Join(types, ", ")
}

func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	}
	return false
}

func lookupFold(values map[string]any, name string) (any, bool) {
	if v, ok := values[name]; ok {
		return v, true
	}
	for k, v := range values {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return nil, false
}

// stringValue renders a parameter value: strings as is, other scalars in
// their JSON form, objects and arrays as JSON.
func stringValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return jsonString(v)
	}
}

func jsonString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func stringSlice(items []any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, stringValue(item))
	}
	return result
}

func escapeAll(items []string, escape func(string) string) []string {
	for i, item := range items {
		items[i] = escape(item)
	}
	return items
}

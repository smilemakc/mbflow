package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const rcServiceAccountJSON = `{"type":"service_account","project_id":"rc","client_email":"rc@rc.iam.gserviceaccount.com","private_key_id":"1","private_key":"-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----\n","token_uri":"http://127.0.0.1:1/token"}`

// rcGoogleServer is a fake Google API: each handler is matched by
// "METHOD suffix" against the request path.
type rcGoogleServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []rcGoogleRequest
}

type rcGoogleRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
}

func rcNewGoogleServer(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) *rcGoogleServer {
	t.Helper()
	gs := &rcGoogleServer{}
	gs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(data, &body)
		gs.mu.Lock()
		gs.requests = append(gs.requests, rcGoogleRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body})
		gs.mu.Unlock()
		for key, h := range routes {
			parts := strings.SplitN(key, " ", 2)
			if r.Method == parts[0] && strings.Contains(r.URL.Path, parts[1]) {
				w.Header().Set("Content-Type", "application/json")
				h(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"message":"no route"}}`))
	}))
	t.Cleanup(gs.Close)
	return gs
}

func rcJSON(body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }
}

func rcStatus(code int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"error":{"code":500,"message":"backend failure"}}`))
	}
}

func rcSheetsService(t *testing.T, gs *rcGoogleServer) *sheets.Service {
	t.Helper()
	srv, err := sheets.NewService(context.Background(), option.WithEndpoint(gs.URL+"/"), option.WithHTTPClient(gs.Client()))
	require.NoError(t, err)
	return srv
}

func rcDriveService(t *testing.T, gs *rcGoogleServer) *drive.Service {
	t.Helper()
	srv, err := drive.NewService(context.Background(), option.WithEndpoint(gs.URL+"/"), option.WithHTTPClient(gs.Client()))
	require.NoError(t, err)
	return srv
}

// --- Google Sheets ---

func TestRC_GoogleSheets_ExecuteRead(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /values/": rcJSON(`{"range":"Sheet1!A1:B2","majorDimension":"ROWS","values":[["a","b"],["c","d"]]}`),
	})
	e := NewGoogleSheetsExecutor()
	out, err := e.executeRead(context.Background(), rcSheetsService(t, gs), "sid", "Sheet1!A1:B2", "ROWS")
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 2, out.RowCount)
	assert.Equal(t, 2, out.ColumnCount)
	assert.Equal(t, "Sheet1!A1:B2", out.Metadata["range"])
	assert.Contains(t, gs.requests[0].Path, "/spreadsheets/sid/values/")
	assert.Contains(t, gs.requests[0].Query, "majorDimension=ROWS")

	empty := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /values/": rcJSON(`{"range":"A1"}`),
	})
	out, err = e.executeRead(context.Background(), rcSheetsService(t, empty), "sid", "A1", "ROWS")
	require.NoError(t, err)
	assert.Equal(t, 0, out.RowCount)
	assert.Equal(t, 0, out.ColumnCount)

	failing := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){"GET /values/": rcStatus(500)})
	_, err = e.executeRead(context.Background(), rcSheetsService(t, failing), "sid", "A1", "ROWS")
	require.ErrorContains(t, err, "failed to read from spreadsheet")
}

func TestRC_GoogleSheets_ExecuteWrite(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"PUT /values/": rcJSON(`{"updatedCells":4,"updatedRows":2,"updatedColumns":2,"updatedRange":"Data!A1:B2"}`),
	})
	e := NewGoogleSheetsExecutor()
	input := []any{
		map[string]any{"name": " Alice ", "age": 30},
		map[string]any{"name": "Bob", "tags": []any{"x"}},
	}
	out, err := e.executeWrite(context.Background(), rcSheetsService(t, gs), "sid", "Data", "Z9", input,
		"RAW", "ROWS", []string{"name", "age"})
	require.NoError(t, err)
	assert.Equal(t, 4, out.UpdatedCells)
	assert.Equal(t, 2, out.UpdatedRows)
	assert.Equal(t, "Data!A1:B2", out.UpdatedRange)
	assert.Equal(t, int64(2), out.Metadata["updated_columns"])

	req := gs.requests[0]
	assert.Contains(t, req.Path, "Data!A1:B2", "user range is replaced by the computed one")
	assert.Contains(t, req.Query, "valueInputOption=RAW")
	assert.Equal(t, []any{[]any{"Alice", float64(30)}, []any{"Bob", ""}}, req.Body["values"])

	_, err = e.executeWrite(context.Background(), rcSheetsService(t, gs), "sid", "", "", []any{}, "RAW", "ROWS", nil)
	require.ErrorContains(t, err, "no data to write")

	_, err = e.executeWrite(context.Background(), rcSheetsService(t, gs), "sid", "", "", 42, "RAW", "ROWS", nil)
	require.ErrorContains(t, err, "unsupported input type")

	failing := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){"PUT /values/": rcStatus(500)})
	_, err = e.executeWrite(context.Background(), rcSheetsService(t, failing), "sid", "S", "", [][]any{{1, 2, 3}}, "RAW", "ROWS", nil)
	require.ErrorContains(t, err, "failed to write to spreadsheet (range: S!A1:C1, rows: 1, cols: 3)")
}

func TestRC_GoogleSheets_ExecuteAppend(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST :append": rcJSON(`{"updates":{"updatedCells":3,"updatedRows":1,"updatedColumns":3,"updatedRange":"S!A5:C5"}}`),
	})
	e := NewGoogleSheetsExecutor()
	out, err := e.executeAppend(context.Background(), rcSheetsService(t, gs), "sid", "S", "", `[[1,2,3]]`, "USER_ENTERED", "ROWS", nil)
	require.NoError(t, err)
	assert.Equal(t, 3, out.UpdatedCells)
	assert.Equal(t, "S!A5:C5", out.UpdatedRange)
	assert.Contains(t, gs.requests[0].Path, "S!A:C")
	assert.Contains(t, gs.requests[0].Query, "insertDataOption=INSERT_ROWS")

	// COLUMNS mode: the number of inner arrays is the number of columns.
	_, err = e.executeAppend(context.Background(), rcSheetsService(t, gs), "sid", "", "", [][]any{{1}, {2}}, "RAW", "COLUMNS", nil)
	require.NoError(t, err)
	assert.Contains(t, gs.requests[1].Path, "A:B")

	_, err = e.executeAppend(context.Background(), rcSheetsService(t, gs), "sid", "", "", map[string]any{"rows": []any{}}, "RAW", "ROWS", nil)
	require.ErrorContains(t, err, "no data to append")

	_, err = e.executeAppend(context.Background(), rcSheetsService(t, gs), "sid", "", "", "not json", "RAW", "ROWS", nil)
	require.ErrorContains(t, err, "failed to parse JSON from string input")

	failing := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){"POST :append": rcStatus(500)})
	_, err = e.executeAppend(context.Background(), rcSheetsService(t, failing), "sid", "", "", [][]any{{1}}, "RAW", "ROWS", nil)
	require.ErrorContains(t, err, "failed to append to spreadsheet")
}

func TestRC_GoogleSheets_ExecuteConfigErrors(t *testing.T) {
	e := NewGoogleSheetsExecutor()
	ctx := context.Background()

	_, err := e.Execute(ctx, map[string]any{}, nil)
	require.Error(t, err)
	_, err = e.Execute(ctx, map[string]any{"operation": "read"}, nil)
	require.Error(t, err)
	_, err = e.Execute(ctx, map[string]any{"operation": "read", "spreadsheet_id": "x"}, nil)
	require.ErrorContains(t, err, "credentials are required")
	_, err = e.Execute(ctx, map[string]any{"operation": "read", "spreadsheet_id": "x", "credentials": "{bad"}, nil)
	require.ErrorContains(t, err, "failed to create sheets service")
	_, err = e.Execute(ctx, map[string]any{"operation": "delete", "spreadsheet_id": "x", "credentials": rcServiceAccountJSON}, nil)
	require.ErrorContains(t, err, "unsupported operation: delete")
}

func TestRC_GoogleSheets_ExtractValuesEdgeCases(t *testing.T) {
	e := NewGoogleSheetsExecutor()

	rows, err := e.extractValuesFromInput([]map[string]any{{"b": 2, "a": 1}}, nil)
	require.NoError(t, err)
	assert.Equal(t, [][]any{{1, 2}}, rows)

	rows, err = e.extractValuesFromInput([]byte(`[{"x":"1"}]`), nil)
	require.NoError(t, err)
	assert.Equal(t, [][]any{{"1"}}, rows)

	_, err = e.extractValuesFromInput([]byte(`{`), nil)
	require.ErrorContains(t, err, "failed to parse JSON from bytes input")

	_, err = e.extractValuesFromInput([]any{1}, nil)
	require.ErrorContains(t, err, "unsupported row type at index 0")

	rows, err = e.extractValuesFromInput(map[string]any{"json": `[[1,2]]`}, nil)
	require.NoError(t, err)
	assert.Equal(t, [][]any{{float64(1), float64(2)}}, rows)

	rows, err = e.extractValuesFromInput(map[string]any{"text": `not json`, "k": "v"}, []string{"k"})
	require.NoError(t, err)
	assert.Equal(t, [][]any{{"v"}}, rows)

	assert.Equal(t, 1, e.maxColumns([][]any{{}}))
	assert.Equal(t, 3, e.maxColumns([][]any{{1}, {1, 2, 3}}))
}

func TestRC_GoogleSheets_SerializeValueTypes(t *testing.T) {
	e := NewGoogleSheetsExecutor()
	type pt struct{ X int }
	assert.Equal(t, uint8(5), e.serializeValue(uint8(5)))
	assert.Equal(t, int16(5), e.serializeValue(int16(5)))
	assert.Equal(t, float32(1.5), e.serializeValue(float32(1.5)))
	assert.Equal(t, `{"X":1}`, e.serializeValue(pt{X: 1}))
	assert.Equal(t, "plain", e.serializeValue(json.RawMessage(`"plain"`)))
	assert.Equal(t, "[x]", strings.ReplaceAll(e.serializeValue([]string{"x"}).(string), `"`, ""))
	ch := make(chan int)
	assert.Contains(t, e.serializeValue(ch), "0x")
	assert.Contains(t, e.serializeValue(map[string]any{"c": ch}), "map[")
}

func TestRC_GoogleSheets_ExtractColumnsVariants(t *testing.T) {
	e := NewGoogleSheetsExecutor()
	assert.Nil(t, e.extractColumns(map[string]any{"columns": ""}))
	assert.Equal(t, []string{"a", "b"}, e.extractColumns(map[string]any{"columns": " a , ,b "}))
	assert.Equal(t, []string{"x"}, e.extractColumns(map[string]any{"columns": []string{"x"}}))
	assert.Equal(t, []string{"y"}, e.extractColumns(map[string]any{"columns": []any{"y", "", 1}}))
	assert.Nil(t, e.extractColumns(map[string]any{"columns": 5}))
}

// --- Google Drive ---

func TestRC_GoogleDrive_CreateSpreadsheetAndFolder(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /files": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id":"new-id","name":"Report","mimeType":"application/vnd.google-apps.spreadsheet","webViewLink":"https://docs/x","createdTime":"2026-01-01T00:00:00Z"}`))
		},
	})
	e := NewGoogleDriveExecutor()
	srv := rcDriveService(t, gs)

	out, err := e.executeCreateSpreadsheet(context.Background(), srv, map[string]any{"file_name": "Report", "parent_folder_id": "p1"})
	require.NoError(t, err)
	assert.Equal(t, "new-id", out.FileID)
	assert.Equal(t, "https://docs/x", out.WebViewURL)
	assert.Equal(t, "2026-01-01T00:00:00Z", out.Metadata["created_time"])
	assert.Equal(t, "application/vnd.google-apps.spreadsheet", gs.requests[0].Body["mimeType"])
	assert.Equal(t, []any{"p1"}, gs.requests[0].Body["parents"])

	out, err = e.executeCreateFolder(context.Background(), srv, map[string]any{"parent_folder_id": "p2"})
	require.NoError(t, err)
	assert.Equal(t, "new-id", out.FolderID)
	assert.Equal(t, "Untitled Folder", gs.requests[1].Body["name"])
	assert.Equal(t, "application/vnd.google-apps.folder", gs.requests[1].Body["mimeType"])

	out, err = e.executeCreateSpreadsheet(context.Background(), srv, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "Untitled Spreadsheet", gs.requests[2].Body["name"])
	assert.NotContains(t, gs.requests[2].Body, "parents")
	_ = out

	failing := rcDriveService(t, rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){"POST /files": rcStatus(500)}))
	_, err = e.executeCreateSpreadsheet(context.Background(), failing, map[string]any{})
	require.ErrorContains(t, err, "failed to create spreadsheet")
	_, err = e.executeCreateFolder(context.Background(), failing, map[string]any{})
	require.ErrorContains(t, err, "failed to create folder")
}

func TestRC_GoogleDrive_ListFiles(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /files": rcJSON(`{"files":[{"id":"1","name":"a","size":"10"},{"id":"2","name":"b"}]}`),
	})
	e := NewGoogleDriveExecutor()
	out, err := e.executeListFiles(context.Background(), rcDriveService(t, gs), map[string]any{"parent_folder_id": "folder", "max_results": 5})
	require.NoError(t, err)
	assert.Equal(t, 2, out.FileCount)
	assert.Equal(t, int64(10), out.Files[0]["size"])
	assert.NotContains(t, out.Files[1], "size")
	assert.Equal(t, "'folder' in parents and trashed=false", out.Metadata["query"])
	assert.Contains(t, gs.requests[0].Query, "pageSize=5")

	out, err = e.executeListFiles(context.Background(), rcDriveService(t, gs), map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "trashed=false", out.Metadata["query"])

	failing := rcDriveService(t, rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){"GET /files": rcStatus(500)}))
	_, err = e.executeListFiles(context.Background(), failing, map[string]any{})
	require.ErrorContains(t, err, "failed to list files")
}

func TestRC_GoogleDrive_Delete(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /files/f1":    rcJSON(`{"id":"f1","name":"doc","mimeType":"text/plain"}`),
		"DELETE /files/f1": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
		"GET /files/f2":    rcJSON(`{"id":"f2","name":"locked"}`),
		"DELETE /files/f2": rcStatus(500),
	})
	e := NewGoogleDriveExecutor()
	srv := rcDriveService(t, gs)

	out, err := e.executeDelete(context.Background(), srv, map[string]any{"file_id": "f1"})
	require.NoError(t, err)
	assert.Equal(t, "doc", out.FileName)
	assert.Equal(t, "text/plain", out.MimeType)

	_, err = e.executeDelete(context.Background(), srv, map[string]any{})
	require.ErrorContains(t, err, "file_id is required for delete")
	_, err = e.executeDelete(context.Background(), srv, map[string]any{"file_id": "missing"})
	require.ErrorContains(t, err, "failed to get file info")
	_, err = e.executeDelete(context.Background(), srv, map[string]any{"file_id": "f2"})
	require.ErrorContains(t, err, "failed to delete file")
}

func TestRC_GoogleDrive_Move(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /files/f1":   rcJSON(`{"parents":["p1","p2"]}`),
		"PATCH /files/f1": rcJSON(`{"id":"f1","name":"doc","mimeType":"text/plain","webViewLink":"l"}`),
		"GET /files/f2":   rcJSON(`{}`),
		"PATCH /files/f2": rcStatus(500),
	})
	e := NewGoogleDriveExecutor()
	srv := rcDriveService(t, gs)

	out, err := e.executeMove(context.Background(), srv, map[string]any{"file_id": "f1", "destination_folder_id": "dst"})
	require.NoError(t, err)
	assert.Equal(t, "dst", out.DestinationFileID)
	assert.Equal(t, []string{"p1", "p2"}, out.Metadata["previous_parents"])
	var patch rcGoogleRequest
	for _, r := range gs.requests {
		if r.Method == http.MethodPatch {
			patch = r
		}
	}
	assert.Contains(t, patch.Query, "addParents=dst")
	assert.Contains(t, patch.Query, "removeParents=p1%2Cp2")

	_, err = e.executeMove(context.Background(), srv, map[string]any{})
	require.ErrorContains(t, err, "file_id is required for move")
	_, err = e.executeMove(context.Background(), srv, map[string]any{"file_id": "f1"})
	require.ErrorContains(t, err, "destination_folder_id is required")
	_, err = e.executeMove(context.Background(), srv, map[string]any{"file_id": "nope", "destination_folder_id": "d"})
	require.ErrorContains(t, err, "failed to get file info")
	_, err = e.executeMove(context.Background(), srv, map[string]any{"file_id": "f2", "destination_folder_id": "d"})
	require.ErrorContains(t, err, "failed to move file")
}

func TestRC_GoogleDrive_Copy(t *testing.T) {
	gs := rcNewGoogleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /files/f1/copy": rcJSON(`{"id":"c1","name":"Copy","mimeType":"text/plain","webViewLink":"l"}`),
		"POST /files/f2/copy": rcStatus(500),
	})
	e := NewGoogleDriveExecutor()
	srv := rcDriveService(t, gs)

	out, err := e.executeCopy(context.Background(), srv, map[string]any{"file_id": "f1", "file_name": "Copy", "destination_folder_id": "d"})
	require.NoError(t, err)
	assert.Equal(t, "c1", out.FileID)
	assert.Equal(t, "f1", out.SourceFileID)
	assert.Equal(t, "Copy", gs.requests[0].Body["name"])
	assert.Equal(t, []any{"d"}, gs.requests[0].Body["parents"])

	_, err = e.executeCopy(context.Background(), srv, map[string]any{})
	require.ErrorContains(t, err, "file_id is required for copy")
	_, err = e.executeCopy(context.Background(), srv, map[string]any{"file_id": "f2"})
	require.ErrorContains(t, err, "failed to copy file")
}

func TestRC_GoogleDrive_ExecuteDispatchErrors(t *testing.T) {
	e := NewGoogleDriveExecutor()
	ctx := context.Background()
	_, err := e.Execute(ctx, map[string]any{}, nil)
	require.Error(t, err)
	_, err = e.Execute(ctx, map[string]any{"operation": "list_files"}, nil)
	require.ErrorContains(t, err, "credentials are required")
	_, err = e.Execute(ctx, map[string]any{"operation": "list_files", "credentials": "{bad"}, nil)
	require.ErrorContains(t, err, "failed to create drive service")
	_, err = e.Execute(ctx, map[string]any{"operation": "rename", "credentials": rcServiceAccountJSON}, nil)
	require.ErrorContains(t, err, "unsupported operation: rename")
	// Operations whose required fields are missing fail before any API call.
	for _, op := range []string{"delete", "move", "copy"} {
		_, err = e.Execute(ctx, map[string]any{"operation": op, "credentials": rcServiceAccountJSON}, nil)
		require.ErrorContains(t, err, "file_id is required", op)
	}
}

package builtin

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGoogleDriveExecutor_Validate(t *testing.T) {
	executor := NewGoogleDriveExecutor()

	// Valid credentials JSON
	validCreds := `{"type":"service_account","project_id":"test"}`

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid create_spreadsheet config",
			config: map[string]any{
				"operation":   "create_spreadsheet",
				"credentials": validCreds,
				"file_name":   "Test Spreadsheet",
			},
			wantErr: false,
		},
		{
			name: "valid create_folder config",
			config: map[string]any{
				"operation":   "create_folder",
				"credentials": validCreds,
				"folder_name": "Test Folder",
			},
			wantErr: false,
		},
		{
			name: "valid list_files config",
			config: map[string]any{
				"operation":   "list_files",
				"credentials": validCreds,
			},
			wantErr: false,
		},
		{
			name: "valid delete config",
			config: map[string]any{
				"operation":   "delete",
				"credentials": validCreds,
				"file_id":     "1234567890",
			},
			wantErr: false,
		},
		{
			name: "valid move config",
			config: map[string]any{
				"operation":             "move",
				"credentials":           validCreds,
				"file_id":               "1234567890",
				"destination_folder_id": "folder123",
			},
			wantErr: false,
		},
		{
			name: "valid copy config",
			config: map[string]any{
				"operation":   "copy",
				"credentials": validCreds,
				"file_id":     "1234567890",
			},
			wantErr: false,
		},
		{
			name: "missing operation",
			config: map[string]any{
				"credentials": validCreds,
			},
			wantErr: true,
			errMsg:  "operation",
		},
		{
			name: "missing credentials",
			config: map[string]any{
				"operation": "list_files",
			},
			wantErr: true,
			errMsg:  "credentials",
		},
		{
			name: "invalid operation",
			config: map[string]any{
				"operation":   "invalid_op",
				"credentials": validCreds,
			},
			wantErr: true,
			errMsg:  "invalid operation",
		},
		{
			name: "invalid credentials JSON",
			config: map[string]any{
				"operation":   "list_files",
				"credentials": "not valid json",
			},
			wantErr: true,
			errMsg:  "must be valid JSON",
		},
		{
			name: "delete missing file_id",
			config: map[string]any{
				"operation":   "delete",
				"credentials": validCreds,
			},
			wantErr: true,
			errMsg:  "file_id is required",
		},
		{
			name: "move missing file_id",
			config: map[string]any{
				"operation":             "move",
				"credentials":           validCreds,
				"destination_folder_id": "folder123",
			},
			wantErr: true,
			errMsg:  "file_id is required",
		},
		{
			name: "move missing destination_folder_id",
			config: map[string]any{
				"operation":   "move",
				"credentials": validCreds,
				"file_id":     "1234567890",
			},
			wantErr: true,
			errMsg:  "destination_folder_id is required",
		},
		{
			name: "copy missing file_id",
			config: map[string]any{
				"operation":   "copy",
				"credentials": validCreds,
			},
			wantErr: true,
			errMsg:  "file_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing '%s', got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = '%v', should contain '%s'", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGoogleDriveExecutor_Execute_InvalidConfig(t *testing.T) {
	executor := NewGoogleDriveExecutor()
	ctx := context.Background()

	tests := []struct {
		name    string
		config  map[string]any
		input   any
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing operation",
			config: map[string]any{
				"credentials": `{"type":"service_account"}`,
			},
			wantErr: true,
			errMsg:  "operation",
		},
		{
			name: "missing credentials",
			config: map[string]any{
				"operation": "list_files",
			},
			wantErr: true,
			errMsg:  "credentials",
		},
		{
			name: "invalid operation",
			config: map[string]any{
				"operation":   "invalid_operation",
				"credentials": `{"type":"service_account"}`,
			},
			wantErr: true,
			errMsg:  "unsupported operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.Execute(ctx, tt.config, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() expected error containing '%s', got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Execute() error = '%v', should contain '%s'", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGoogleDriveExecutor_OutputStructure(t *testing.T) {
	// Test that output can be marshaled to JSON correctly
	output := &GoogleDriveOutput{
		Success:    true,
		Operation:  "list_files",
		FileCount:  2,
		DurationMs: 150,
		Files: []map[string]any{
			{
				"id":        "file1",
				"name":      "test1.txt",
				"mime_type": "text/plain",
			},
			{
				"id":        "file2",
				"name":      "test2.pdf",
				"mime_type": "application/pdf",
			},
		},
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal output: %v", err)
	}

	var decoded GoogleDriveOutput
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	if decoded.Success != output.Success {
		t.Errorf("Success mismatch: got %v, want %v", decoded.Success, output.Success)
	}
	if decoded.FileCount != output.FileCount {
		t.Errorf("FileCount mismatch: got %d, want %d", decoded.FileCount, output.FileCount)
	}
	if len(decoded.Files) != len(output.Files) {
		t.Errorf("Files length mismatch: got %d, want %d", len(decoded.Files), len(output.Files))
	}
}

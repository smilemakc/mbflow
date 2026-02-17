package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveExecutor executes operations with Google Drive API.
// Supports: create_spreadsheet, create_folder, list_files, delete, move, copy operations.
type GoogleDriveExecutor struct {
	*executor.BaseExecutor
}

// NewGoogleDriveExecutor creates a new Google Drive executor.
func NewGoogleDriveExecutor() *GoogleDriveExecutor {
	return &GoogleDriveExecutor{
		BaseExecutor: executor.NewBaseExecutor("google_drive"),
	}
}

// GoogleDriveOutput represents the output structure.
type GoogleDriveOutput struct {
	Success           bool             `json:"success"`
	Operation         string           `json:"operation"`
	FileID            string           `json:"file_id,omitempty"`
	FolderID          string           `json:"folder_id,omitempty"`
	FileName          string           `json:"file_name,omitempty"`
	MimeType          string           `json:"mime_type,omitempty"`
	WebViewURL        string           `json:"web_view_url,omitempty"`
	Files             []map[string]any `json:"files,omitempty"`
	FileCount         int              `json:"file_count,omitempty"`
	SourceFileID      string           `json:"source_file_id,omitempty"`
	DestinationFileID string           `json:"destination_file_id,omitempty"`
	Metadata          map[string]any   `json:"metadata,omitempty"`
	DurationMs        int64            `json:"duration_ms"`
}

// Execute implements the Executor interface.
func (e *GoogleDriveExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Extract required config
	operation, err := e.GetString(config, "operation")
	if err != nil {
		return nil, err
	}

	credentialsJSON, err := e.GetString(config, "credentials")
	if err != nil {
		return nil, fmt.Errorf("credentials are required: %w", err)
	}

	// Create Google Drive service
	srv, err := e.createDriveService(ctx, credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	var output *GoogleDriveOutput

	switch operation {
	case "create_spreadsheet":
		output, err = e.executeCreateSpreadsheet(ctx, srv, config)
	case "create_folder":
		output, err = e.executeCreateFolder(ctx, srv, config)
	case "list_files":
		output, err = e.executeListFiles(ctx, srv, config)
	case "delete":
		output, err = e.executeDelete(ctx, srv, config)
	case "move":
		output, err = e.executeMove(ctx, srv, config)
	case "copy":
		output, err = e.executeCopy(ctx, srv, config)
	default:
		return nil, fmt.Errorf("unsupported operation: %s (supported: create_spreadsheet, create_folder, list_files, delete, move, copy)", operation)
	}

	if err != nil {
		return nil, err
	}

	output.Operation = operation
	output.DurationMs = time.Since(startTime).Milliseconds()

	return output, nil
}

// createDriveService creates a Google Drive API service using service account credentials.
func (e *GoogleDriveExecutor) createDriveService(ctx context.Context, credentialsJSON string) (*drive.Service, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(credentialsJSON), drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	srv, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return srv, nil
}

// executeCreateSpreadsheet creates a new Google Sheets spreadsheet.
func (e *GoogleDriveExecutor) executeCreateSpreadsheet(ctx context.Context, srv *drive.Service, config map[string]any) (*GoogleDriveOutput, error) {
	fileName := e.GetStringDefault(config, "file_name", "Untitled Spreadsheet")
	parentFolderID := e.GetStringDefault(config, "parent_folder_id", "")

	// Create a new Google Sheets file
	file := &drive.File{
		Name:     fileName,
		MimeType: "application/vnd.google-apps.spreadsheet",
	}

	if parentFolderID != "" {
		file.Parents = []string{parentFolderID}
	}

	createdFile, err := srv.Files.Create(file).Fields("id, name, mimeType, webViewLink, createdTime").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create spreadsheet: %w", err)
	}

	return &GoogleDriveOutput{
		Success:    true,
		FileID:     createdFile.Id,
		FileName:   createdFile.Name,
		MimeType:   createdFile.MimeType,
		WebViewURL: createdFile.WebViewLink,
		Metadata: map[string]any{
			"created_time": createdFile.CreatedTime,
		},
	}, nil
}

// executeCreateFolder creates a new folder in Google Drive.
func (e *GoogleDriveExecutor) executeCreateFolder(ctx context.Context, srv *drive.Service, config map[string]any) (*GoogleDriveOutput, error) {
	folderName := e.GetStringDefault(config, "folder_name", "Untitled Folder")
	parentFolderID := e.GetStringDefault(config, "parent_folder_id", "")

	// Create a new folder
	file := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	if parentFolderID != "" {
		file.Parents = []string{parentFolderID}
	}

	createdFile, err := srv.Files.Create(file).Fields("id, name, mimeType, webViewLink, createdTime").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return &GoogleDriveOutput{
		Success:    true,
		FolderID:   createdFile.Id,
		FileName:   createdFile.Name,
		MimeType:   createdFile.MimeType,
		WebViewURL: createdFile.WebViewLink,
		Metadata: map[string]any{
			"created_time": createdFile.CreatedTime,
		},
	}, nil
}

// executeListFiles lists files in a folder (or root if no folder specified).
func (e *GoogleDriveExecutor) executeListFiles(ctx context.Context, srv *drive.Service, config map[string]any) (*GoogleDriveOutput, error) {
	parentFolderID := e.GetStringDefault(config, "parent_folder_id", "")
	maxResults := e.GetIntDefault(config, "max_results", 100)
	orderBy := e.GetStringDefault(config, "order_by", "modifiedTime desc")

	// Build query
	query := "trashed=false"
	if parentFolderID != "" {
		query = fmt.Sprintf("'%s' in parents and trashed=false", parentFolderID)
	}

	fileList, err := srv.Files.List().
		Q(query).
		PageSize(int64(maxResults)).
		OrderBy(orderBy).
		Fields("files(id, name, mimeType, createdTime, modifiedTime, size, webViewLink)").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Convert files to map format
	files := make([]map[string]any, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		fileMap := map[string]any{
			"id":            f.Id,
			"name":          f.Name,
			"mime_type":     f.MimeType,
			"created_time":  f.CreatedTime,
			"modified_time": f.ModifiedTime,
			"web_view_link": f.WebViewLink,
		}
		if f.Size > 0 {
			fileMap["size"] = f.Size
		}
		files = append(files, fileMap)
	}

	return &GoogleDriveOutput{
		Success:   true,
		Files:     files,
		FileCount: len(files),
		Metadata: map[string]any{
			"parent_folder_id": parentFolderID,
			"query":            query,
		},
	}, nil
}

// executeDelete deletes a file or folder by ID.
func (e *GoogleDriveExecutor) executeDelete(ctx context.Context, srv *drive.Service, config map[string]any) (*GoogleDriveOutput, error) {
	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required for delete: %w", err)
	}

	// Get file info before deletion for response
	file, err := srv.Files.Get(fileID).Fields("id, name, mimeType").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	err = srv.Files.Delete(fileID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}

	return &GoogleDriveOutput{
		Success:  true,
		FileID:   fileID,
		FileName: file.Name,
		MimeType: file.MimeType,
	}, nil
}

// executeMove moves a file to a different folder.
func (e *GoogleDriveExecutor) executeMove(ctx context.Context, srv *drive.Service, config map[string]any) (*GoogleDriveOutput, error) {
	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required for move: %w", err)
	}

	destinationFolderID, err := e.GetString(config, "destination_folder_id")
	if err != nil {
		return nil, fmt.Errorf("destination_folder_id is required for move: %w", err)
	}

	// Get current parents
	file, err := srv.Files.Get(fileID).Fields("parents").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Build previous parents string
	var previousParents string
	if len(file.Parents) > 0 {
		previousParents = file.Parents[0]
		for i := 1; i < len(file.Parents); i++ {
			previousParents += "," + file.Parents[i]
		}
	}

	// Move file
	updatedFile, err := srv.Files.Update(fileID, nil).
		AddParents(destinationFolderID).
		RemoveParents(previousParents).
		Fields("id, name, mimeType, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to move file: %w", err)
	}

	return &GoogleDriveOutput{
		Success:           true,
		FileID:            updatedFile.Id,
		FileName:          updatedFile.Name,
		MimeType:          updatedFile.MimeType,
		WebViewURL:        updatedFile.WebViewLink,
		DestinationFileID: destinationFolderID,
		Metadata: map[string]any{
			"previous_parents": file.Parents,
		},
	}, nil
}

// executeCopy copies a file to a different location.
func (e *GoogleDriveExecutor) executeCopy(ctx context.Context, srv *drive.Service, config map[string]any) (*GoogleDriveOutput, error) {
	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required for copy: %w", err)
	}

	destinationFolderID := e.GetStringDefault(config, "destination_folder_id", "")
	newFileName := e.GetStringDefault(config, "file_name", "")

	// Build copy request
	copiedFile := &drive.File{}

	if newFileName != "" {
		copiedFile.Name = newFileName
	}

	if destinationFolderID != "" {
		copiedFile.Parents = []string{destinationFolderID}
	}

	// Copy file
	result, err := srv.Files.Copy(fileID, copiedFile).Fields("id, name, mimeType, webViewLink").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	return &GoogleDriveOutput{
		Success:           true,
		SourceFileID:      fileID,
		FileID:            result.Id,
		FileName:          result.Name,
		MimeType:          result.MimeType,
		WebViewURL:        result.WebViewLink,
		DestinationFileID: destinationFolderID,
		Metadata: map[string]any{
			"original_file_id": fileID,
		},
	}, nil
}

// Validate validates the executor configuration.
func (e *GoogleDriveExecutor) Validate(config map[string]any) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "operation", "credentials"); err != nil {
		return err
	}

	// Validate operation
	operation, err := e.GetString(config, "operation")
	if err != nil {
		return err
	}

	validOperations := map[string]bool{
		"create_spreadsheet": true,
		"create_folder":      true,
		"list_files":         true,
		"delete":             true,
		"move":               true,
		"copy":               true,
	}

	if !validOperations[operation] {
		return fmt.Errorf("invalid operation: %s (supported: create_spreadsheet, create_folder, list_files, delete, move, copy)", operation)
	}

	// Validate credentials (must be valid JSON)
	credentials, err := e.GetString(config, "credentials")
	if err != nil {
		return err
	}
	if credentials == "" {
		return fmt.Errorf("credentials cannot be empty")
	}

	// Try to parse credentials as JSON
	var creds map[string]any
	if err := json.Unmarshal([]byte(credentials), &creds); err != nil {
		return fmt.Errorf("credentials must be valid JSON: %w", err)
	}

	// Operation-specific validation
	switch operation {
	case "delete":
		if _, err := e.GetString(config, "file_id"); err != nil {
			return fmt.Errorf("file_id is required for delete operation")
		}
	case "move":
		if _, err := e.GetString(config, "file_id"); err != nil {
			return fmt.Errorf("file_id is required for move operation")
		}
		if _, err := e.GetString(config, "destination_folder_id"); err != nil {
			return fmt.Errorf("destination_folder_id is required for move operation")
		}
	case "copy":
		if _, err := e.GetString(config, "file_id"); err != nil {
			return fmt.Errorf("file_id is required for copy operation")
		}
	}

	return nil
}

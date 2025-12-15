package filestorage

import (
	"testing"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// File signatures for MIME detection tests
var (
	pngSignature  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	jpegSignature = []byte{0xFF, 0xD8, 0xFF, 0xE0}
	gifSignature  = []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61} // GIF89a
	pdfSignature  = []byte{0x25, 0x50, 0x44, 0x46}             // %PDF
	zipSignature  = []byte{0x50, 0x4B, 0x03, 0x04}             // PK..
)

// ============== A. Initialization Tests ==============

func TestMimeValidator_New_DefaultTypes(t *testing.T) {
	validator := NewMimeValidator()
	require.NotNil(t, validator)

	// Verify some default types are allowed
	assert.True(t, validator.IsAllowed("image/png"))
	assert.True(t, validator.IsAllowed("application/pdf"))
	assert.True(t, validator.IsAllowed("text/plain"))
}

func TestMimeValidator_NewWithTypes_Custom(t *testing.T) {
	customTypes := []string{
		"image/png",
		"application/pdf",
		"text/plain",
	}

	validator := NewMimeValidatorWithTypes(customTypes)
	require.NotNil(t, validator)

	// Verify custom types are allowed
	assert.True(t, validator.IsAllowed("image/png"))
	assert.True(t, validator.IsAllowed("application/pdf"))
	assert.True(t, validator.IsAllowed("text/plain"))

	// Verify non-custom types are not allowed
	assert.False(t, validator.IsAllowed("image/jpeg"))
	assert.False(t, validator.IsAllowed("video/mp4"))
}

func TestMimeValidator_NewWithTypes_Empty(t *testing.T) {
	validator := NewMimeValidatorWithTypes([]string{})
	require.NotNil(t, validator)

	// All types should be disallowed
	assert.False(t, validator.IsAllowed("image/png"))
	assert.False(t, validator.IsAllowed("text/plain"))
}

func TestMimeValidator_AllowedMimeTypes_Count(t *testing.T) {
	validator := NewMimeValidator()
	types := validator.AllowedMimeTypesList()

	// Should have 36 default types
	// 7 images + 7 documents + 5 audio + 5 video + 6 text + 5 archives = 35
	// Actually count from models.AllowedMimeTypes
	expectedCount := len(models.AllowedMimeTypes)
	assert.Equal(t, expectedCount, len(types))
}

// ============== B. Validation Tests ==============

func TestMimeValidator_Validate_Success(t *testing.T) {
	validator := NewMimeValidator()

	tests := []struct {
		name     string
		mimeType string
	}{
		// Images
		{"image_png", "image/png"},
		{"image_jpeg", "image/jpeg"},
		{"image_gif", "image/gif"},
		{"image_webp", "image/webp"},
		{"image_svg", "image/svg+xml"},
		{"image_bmp", "image/bmp"},
		{"image_tiff", "image/tiff"},

		// Documents
		{"pdf", "application/pdf"},
		{"msword", "application/msword"},
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"excel", "application/vnd.ms-excel"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"powerpoint", "application/vnd.ms-powerpoint"},
		{"pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},

		// Audio
		{"audio_mpeg", "audio/mpeg"},
		{"audio_wav", "audio/wav"},
		{"audio_ogg", "audio/ogg"},
		{"audio_webm", "audio/webm"},
		{"audio_flac", "audio/flac"},

		// Video
		{"video_mp4", "video/mp4"},
		{"video_webm", "video/webm"},
		{"video_ogg", "video/ogg"},
		{"video_mpeg", "video/mpeg"},
		{"video_quicktime", "video/quicktime"},

		// Text
		{"text_plain", "text/plain"},
		{"text_csv", "text/csv"},
		{"text_html", "text/html"},
		{"text_markdown", "text/markdown"},
		{"json", "application/json"},
		{"xml", "application/xml"},

		// Archives
		{"zip", "application/zip"},
		{"gzip", "application/gzip"},
		{"tar", "application/x-tar"},
		{"rar", "application/x-rar-compressed"},
		{"7z", "application/x-7z-compressed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.mimeType)
			assert.NoError(t, err)
			assert.True(t, validator.IsAllowed(tt.mimeType))
		})
	}
}

func TestMimeValidator_Validate_Error_Disallowed(t *testing.T) {
	validator := NewMimeValidator()

	disallowedTypes := []string{
		"application/x-msdownload", // .exe
		"application/x-sh",         // shell script
		"text/x-python",
		"application/octet-stream",
		"video/x-msvideo", // .avi not in whitelist
	}

	for _, mimeType := range disallowedTypes {
		t.Run(mimeType, func(t *testing.T) {
			err := validator.Validate(mimeType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "MIME type not allowed")
			assert.False(t, validator.IsAllowed(mimeType))
		})
	}
}

func TestMimeValidator_Validate_WithCharset(t *testing.T) {
	validator := NewMimeValidator()

	// MIME type with charset parameter should be normalized
	err := validator.Validate("text/plain; charset=utf-8")
	assert.NoError(t, err)

	err = validator.Validate("text/html; charset=iso-8859-1")
	assert.NoError(t, err)

	err = validator.Validate("application/json; charset=utf-8")
	assert.NoError(t, err)
}

func TestMimeValidator_Validate_WithBoundary(t *testing.T) {
	validator := NewMimeValidator()

	// MIME type with boundary parameter (multipart forms) should be normalized
	// Note: Only the base MIME type needs to be in allowlist
	err := validator.Validate("text/plain; boundary=something")
	assert.NoError(t, err)
}

func TestMimeValidator_Validate_CaseInsensitive(t *testing.T) {
	validator := NewMimeValidator()

	// Lowercase
	assert.True(t, validator.IsAllowed("image/png"))

	// Uppercase - Go's MIME handling typically returns lowercase,
	// but let's verify our validator handles it
	// Note: The normalization doesn't change case, so this depends on
	// how AllowedMimeTypes is defined. In our case, it's all lowercase.
	assert.False(t, validator.IsAllowed("IMAGE/PNG"))
}

func TestMimeValidator_Validate_EmptyString(t *testing.T) {
	validator := NewMimeValidator()

	err := validator.Validate("")
	assert.Error(t, err)
	assert.False(t, validator.IsAllowed(""))
}

func TestMimeValidator_Validate_Whitespace(t *testing.T) {
	validator := NewMimeValidator()

	// With leading/trailing spaces - should be normalized
	assert.True(t, validator.IsAllowed(" text/plain "))
}

// ============== C. MIME Detection Tests ==============

func TestDetectMimeType_PNG_Signature(t *testing.T) {
	data := append(pngSignature, []byte("fake png data")...)
	mimeType := DetectMimeType(data)

	assert.Equal(t, "image/png", mimeType)
}

func TestDetectMimeType_JPEG_Signature(t *testing.T) {
	data := append(jpegSignature, []byte("fake jpeg data")...)
	mimeType := DetectMimeType(data)

	// http.DetectContentType returns "image/jpeg" for JPEG
	assert.Contains(t, mimeType, "image/jpeg")
}

func TestDetectMimeType_GIF_Signature(t *testing.T) {
	data := append(gifSignature, []byte("fake gif data")...)
	mimeType := DetectMimeType(data)

	assert.Equal(t, "image/gif", mimeType)
}

func TestDetectMimeType_PDF_Signature(t *testing.T) {
	data := append(pdfSignature, []byte("-1.4\nfake pdf data")...)
	mimeType := DetectMimeType(data)

	assert.Equal(t, "application/pdf", mimeType)
}

func TestDetectMimeType_ZIP_Signature(t *testing.T) {
	data := append(zipSignature, []byte("fake zip data")...)
	mimeType := DetectMimeType(data)

	assert.Equal(t, "application/zip", mimeType)
}

func TestDetectMimeType_JSON_Content(t *testing.T) {
	data := []byte(`{"key": "value", "number": 123}`)
	mimeType := DetectMimeType(data)

	// http.DetectContentType detects JSON as text/plain
	assert.Contains(t, mimeType, "text/plain")
}

func TestDetectMimeType_XML_Content(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><root><item>test</item></root>`)
	mimeType := DetectMimeType(data)

	// http.DetectContentType may detect as text/xml or text/plain
	assert.True(t, mimeType == "text/xml; charset=utf-8" || mimeType == "text/plain; charset=utf-8")
}

func TestDetectMimeType_PlainText(t *testing.T) {
	data := []byte("This is plain text content")
	mimeType := DetectMimeType(data)

	assert.Contains(t, mimeType, "text/plain")
}

func TestDetectMimeType_Binary(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
	mimeType := DetectMimeType(data)

	// Binary data typically detected as application/octet-stream
	assert.Contains(t, mimeType, "application/octet-stream")
}

func TestDetectMimeType_EmptyContent(t *testing.T) {
	data := []byte{}
	mimeType := DetectMimeType(data)

	// Empty content detected as text/plain
	assert.Contains(t, mimeType, "text/plain")
}

func TestDetectMimeType_SmallContent(t *testing.T) {
	data := []byte("Hi")
	mimeType := DetectMimeType(data)

	assert.Contains(t, mimeType, "text/plain")
}

func TestDetectMimeType_HTMLContent(t *testing.T) {
	data := []byte("<!DOCTYPE html><html><body>Test</body></html>")
	mimeType := DetectMimeType(data)

	assert.Contains(t, mimeType, "text/html")
}

// ============== D. Filename Detection Tests ==============

func TestDetectMimeTypeFromFilename_TXT(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("document.txt")
	assert.Contains(t, mimeType, "text/plain")
}

func TestDetectMimeTypeFromFilename_JSON(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("data.json")
	assert.Contains(t, mimeType, "application/json")
}

func TestDetectMimeTypeFromFilename_PDF(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("report.pdf")
	assert.Equal(t, "application/pdf", mimeType)
}

func TestDetectMimeTypeFromFilename_PNG(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("image.png")
	assert.Equal(t, "image/png", mimeType)
}

func TestDetectMimeTypeFromFilename_JPEG(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("photo.jpg")
	assert.Equal(t, "image/jpeg", mimeType)

	mimeType = DetectMimeTypeFromFilename("photo.jpeg")
	assert.Equal(t, "image/jpeg", mimeType)
}

func TestDetectMimeTypeFromFilename_ZIP(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("archive.zip")
	assert.Equal(t, "application/zip", mimeType)
}

func TestDetectMimeTypeFromFilename_MP4(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("video.mp4")
	assert.Contains(t, mimeType, "video/mp4")
}

func TestDetectMimeTypeFromFilename_MP3(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("audio.mp3")
	assert.Contains(t, mimeType, "audio/mpeg")
}

func TestDetectMimeTypeFromFilename_NoExtension(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("README")
	// Should default to application/octet-stream
	assert.Equal(t, "application/octet-stream", mimeType)
}

func TestDetectMimeTypeFromFilename_UnknownExtension(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("file.xyz123")
	// Should default to application/octet-stream
	assert.Equal(t, "application/octet-stream", mimeType)
}

func TestDetectMimeTypeFromFilename_MultipleDotsInName(t *testing.T) {
	mimeType := DetectMimeTypeFromFilename("my.file.name.pdf")
	// Should use the last extension
	assert.Equal(t, "application/pdf", mimeType)
}

func TestDetectMimeTypeFromFilename_CaseSensitivity(t *testing.T) {
	// Extensions are typically case-insensitive in mime package
	mimeType := DetectMimeTypeFromFilename("IMAGE.PNG")
	assert.Equal(t, "image/png", mimeType)

	mimeType = DetectMimeTypeFromFilename("document.PDF")
	assert.Equal(t, "application/pdf", mimeType)
}

// ============== E. Category Tests ==============

func TestGetMimeCategory_Image(t *testing.T) {
	assert.Equal(t, "image", GetMimeCategory("image/png"))
	assert.Equal(t, "image", GetMimeCategory("image/jpeg"))
	assert.Equal(t, "image", GetMimeCategory("image/gif"))
}

func TestGetMimeCategory_Video(t *testing.T) {
	assert.Equal(t, "video", GetMimeCategory("video/mp4"))
	assert.Equal(t, "video", GetMimeCategory("video/webm"))
}

func TestGetMimeCategory_Audio(t *testing.T) {
	assert.Equal(t, "audio", GetMimeCategory("audio/mpeg"))
	assert.Equal(t, "audio", GetMimeCategory("audio/wav"))
}

func TestGetMimeCategory_Application(t *testing.T) {
	assert.Equal(t, "application", GetMimeCategory("application/pdf"))
	assert.Equal(t, "application", GetMimeCategory("application/json"))
}

func TestGetMimeCategory_Text(t *testing.T) {
	assert.Equal(t, "text", GetMimeCategory("text/plain"))
	assert.Equal(t, "text", GetMimeCategory("text/html"))
}

func TestGetMimeCategory_Invalid(t *testing.T) {
	// Empty string returns empty string
	assert.Equal(t, "", GetMimeCategory(""))
	// String without slash returns the whole string as category
	assert.Equal(t, "invalid", GetMimeCategory("invalid"))
}

func TestIsImageMime_True(t *testing.T) {
	assert.True(t, IsImageMime("image/png"))
	assert.True(t, IsImageMime("image/jpeg"))
	assert.True(t, IsImageMime("image/gif"))
	assert.True(t, IsImageMime("image/webp"))
}

func TestIsImageMime_False(t *testing.T) {
	assert.False(t, IsImageMime("video/mp4"))
	assert.False(t, IsImageMime("text/plain"))
	assert.False(t, IsImageMime("application/pdf"))
}

func TestIsVideoMime_True(t *testing.T) {
	assert.True(t, IsVideoMime("video/mp4"))
	assert.True(t, IsVideoMime("video/webm"))
	assert.True(t, IsVideoMime("video/quicktime"))
}

func TestIsVideoMime_False(t *testing.T) {
	assert.False(t, IsVideoMime("image/png"))
	assert.False(t, IsVideoMime("audio/mpeg"))
}

func TestIsAudioMime_True(t *testing.T) {
	assert.True(t, IsAudioMime("audio/mpeg"))
	assert.True(t, IsAudioMime("audio/wav"))
	assert.True(t, IsAudioMime("audio/ogg"))
}

func TestIsAudioMime_False(t *testing.T) {
	assert.False(t, IsAudioMime("video/mp4"))
	assert.False(t, IsAudioMime("text/plain"))
}

func TestIsDocumentMime_PDFs(t *testing.T) {
	assert.True(t, IsDocumentMime("application/pdf"))
}

func TestIsDocumentMime_Office(t *testing.T) {
	assert.True(t, IsDocumentMime("application/msword"))
	// OpenXML formats are detected by "openxmlformats" substring
	assert.True(t, IsDocumentMime("application/vnd.openxmlformats-officedocument.wordprocessingml.document"))
	assert.True(t, IsDocumentMime("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"))
	assert.True(t, IsDocumentMime("application/vnd.openxmlformats-officedocument.presentationml.presentation"))
}

func TestIsDocumentMime_Text(t *testing.T) {
	assert.True(t, IsDocumentMime("text/plain"))
	assert.True(t, IsDocumentMime("text/csv"))
	assert.True(t, IsDocumentMime("text/html"))
	assert.True(t, IsDocumentMime("text/markdown"))
}

func TestIsDocumentMime_Structured(t *testing.T) {
	assert.True(t, IsDocumentMime("application/json"))
	assert.True(t, IsDocumentMime("application/xml"))
}

func TestIsDocumentMime_False(t *testing.T) {
	assert.False(t, IsDocumentMime("image/png"))
	assert.False(t, IsDocumentMime("video/mp4"))
	assert.False(t, IsDocumentMime("audio/mpeg"))
	assert.False(t, IsDocumentMime("application/zip"))
}

// ============== F. Whitelist Management Tests ==============

func TestMimeValidator_AddAllowedType(t *testing.T) {
	validator := NewMimeValidatorWithTypes([]string{"image/png"})

	// Initially only image/png is allowed
	assert.True(t, validator.IsAllowed("image/png"))
	assert.False(t, validator.IsAllowed("image/jpeg"))

	// Add image/jpeg
	validator.AddAllowedType("image/jpeg")

	// Now both should be allowed
	assert.True(t, validator.IsAllowed("image/png"))
	assert.True(t, validator.IsAllowed("image/jpeg"))
}

func TestMimeValidator_RemoveAllowedType(t *testing.T) {
	validator := NewMimeValidatorWithTypes([]string{
		"image/png",
		"image/jpeg",
		"text/plain",
	})

	// Initially all three are allowed
	assert.True(t, validator.IsAllowed("image/png"))
	assert.True(t, validator.IsAllowed("image/jpeg"))
	assert.True(t, validator.IsAllowed("text/plain"))

	// Remove image/jpeg
	validator.RemoveAllowedType("image/jpeg")

	// image/jpeg should not be allowed, others still are
	assert.True(t, validator.IsAllowed("image/png"))
	assert.False(t, validator.IsAllowed("image/jpeg"))
	assert.True(t, validator.IsAllowed("text/plain"))
}

func TestMimeValidator_AllowedMimeTypesList(t *testing.T) {
	types := []string{"image/png", "text/plain", "application/json"}
	validator := NewMimeValidatorWithTypes(types)

	list := validator.AllowedMimeTypesList()

	assert.Len(t, list, 3)
	assert.Contains(t, list, "image/png")
	assert.Contains(t, list, "text/plain")
	assert.Contains(t, list, "application/json")
}

func TestMimeValidator_AllowedMimeTypesList_Empty(t *testing.T) {
	validator := NewMimeValidatorWithTypes([]string{})

	list := validator.AllowedMimeTypesList()

	assert.Len(t, list, 0)
}

// ============== G. Normalization Tests ==============

func TestMimeValidator_Normalize_WithCharset(t *testing.T) {
	validator := NewMimeValidator()

	// These should all normalize to "text/plain"
	assert.True(t, validator.IsAllowed("text/plain"))
	assert.True(t, validator.IsAllowed("text/plain; charset=utf-8"))
	assert.True(t, validator.IsAllowed("text/plain; charset=iso-8859-1"))
	assert.True(t, validator.IsAllowed("text/plain;charset=utf-8")) // No space
}

func TestMimeValidator_Normalize_WithBoundary(t *testing.T) {
	validator := NewMimeValidator()

	assert.True(t, validator.IsAllowed("application/json; boundary=----WebKitFormBoundary"))
	assert.True(t, validator.IsAllowed("text/plain; boundary=something"))
}

func TestMimeValidator_Normalize_MultipleParameters(t *testing.T) {
	validator := NewMimeValidator()

	// Multiple parameters should be stripped
	assert.True(t, validator.IsAllowed("text/html; charset=utf-8; boundary=xyz"))
}

func TestMimeValidator_Normalize_TrailingWhitespace(t *testing.T) {
	validator := NewMimeValidator()

	// Whitespace should be trimmed
	assert.True(t, validator.IsAllowed(" text/plain "))
	assert.True(t, validator.IsAllowed("text/plain "))
	assert.True(t, validator.IsAllowed(" text/plain"))
}

// ============== H. Integration Tests ==============

func TestMimeValidator_IsAllowed_AllDefaultTypes(t *testing.T) {
	validator := NewMimeValidator()

	// Verify all types in models.AllowedMimeTypes are allowed
	for mimeType := range models.AllowedMimeTypes {
		t.Run(mimeType, func(t *testing.T) {
			assert.True(t, validator.IsAllowed(mimeType), "Type %s should be allowed", mimeType)
			assert.NoError(t, validator.Validate(mimeType), "Type %s should validate", mimeType)
		})
	}
}

func TestMimeValidator_Validate_CommonScenarios(t *testing.T) {
	validator := NewMimeValidator()

	scenarios := []struct {
		name     string
		mimeType string
		wantErr  bool
	}{
		{"upload_png_image", "image/png", false},
		{"upload_pdf_document", "application/pdf", false},
		{"upload_json_data", "application/json", false},
		{"upload_video", "video/mp4", false},
		{"upload_archive", "application/zip", false},
		{"reject_executable", "application/x-msdownload", true},
		{"reject_shell_script", "application/x-sh", true},
		{"reject_unknown", "application/custom", true},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			err := validator.Validate(sc.mimeType)
			if sc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

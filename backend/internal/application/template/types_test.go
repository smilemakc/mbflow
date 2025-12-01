package template

import (
	"errors"
	"testing"
)

func TestTemplateError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TemplateError
		wantText string
	}{
		{
			name: "error with path",
			err: &TemplateError{
				Template: "Hello {{env.name}}",
				Variable: "env",
				Path:     "name",
				Err:      ErrVariableNotFound,
			},
			wantText: "template error in 'Hello {{env.name}}': failed to resolve '{{env.name}}': variable not found",
		},
		{
			name: "error without path",
			err: &TemplateError{
				Template: "Hello {{env}}",
				Variable: "env",
				Path:     "",
				Err:      ErrInvalidTemplate,
			},
			wantText: "template error in 'Hello {{env}}': failed to resolve '{{env}}': invalid template syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantText {
				t.Errorf("TemplateError.Error() = %v, want %v", got, tt.wantText)
			}
		})
	}
}

func TestTemplateError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &TemplateError{
		Template: "test",
		Variable: "env",
		Path:     "var",
		Err:      underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

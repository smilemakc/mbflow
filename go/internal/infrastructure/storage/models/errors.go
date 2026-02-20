package models

import "errors"

// Model validation errors
var (
	ErrSelfReferenceEdge = errors.New("edge cannot reference the same node as source and target")
	ErrInvalidStatus     = errors.New("invalid status value")
	ErrInvalidType       = errors.New("invalid type value")
	ErrRequiredField     = errors.New("required field is missing")
	ErrInvalidUUID       = errors.New("invalid UUID format")
)

package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONBMap is a custom type for JSONB columns
type JSONBMap map[string]interface{}

// Value implements the driver.Valuer interface for database serialization
func (j JSONBMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database deserialization
func (j *JSONBMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONBMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONBMap: value is not []byte")
	}

	if len(bytes) == 0 {
		*j = make(JSONBMap)
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Get retrieves a value from the map with type assertion
func (j JSONBMap) Get(key string) (interface{}, bool) {
	val, ok := j[key]
	return val, ok
}

// GetString retrieves a string value from the map
func (j JSONBMap) GetString(key string) string {
	if val, ok := j[key].(string); ok {
		return val
	}
	return ""
}

// GetInt retrieves an int value from the map
func (j JSONBMap) GetInt(key string) int {
	if val, ok := j[key].(float64); ok {
		return int(val)
	}
	return 0
}

// GetFloat retrieves a float64 value from the map
func (j JSONBMap) GetFloat(key string) float64 {
	if val, ok := j[key].(float64); ok {
		return val
	}
	return 0
}

// GetBool retrieves a bool value from the map
func (j JSONBMap) GetBool(key string) bool {
	if val, ok := j[key].(bool); ok {
		return val
	}
	return false
}

// GetMap retrieves a nested map from the map
func (j JSONBMap) GetMap(key string) JSONBMap {
	if val, ok := j[key].(map[string]interface{}); ok {
		return JSONBMap(val)
	}
	return make(JSONBMap)
}

// Set sets a value in the map
func (j JSONBMap) Set(key string, value interface{}) {
	j[key] = value
}

// Delete removes a key from the map
func (j JSONBMap) Delete(key string) {
	delete(j, key)
}

// Has checks if a key exists in the map
func (j JSONBMap) Has(key string) bool {
	_, ok := j[key]
	return ok
}

// Clone creates a deep copy of the map
func (j JSONBMap) Clone() JSONBMap {
	if j == nil {
		return make(JSONBMap)
	}
	bytes, _ := json.Marshal(j)
	var clone JSONBMap
	_ = json.Unmarshal(bytes, &clone)
	return clone
}

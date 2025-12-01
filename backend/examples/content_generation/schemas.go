package main

import (
	"encoding/json"
	"fmt"
)

// qualityAnalysisSchema defines the expected structure for quality analysis output
var qualityAnalysisSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"score": map[string]interface{}{
			"type":        "integer",
			"minimum":     0,
			"maximum":     100,
			"description": "Overall content quality score from 0 to 100",
		},
		"issues": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "List of identified issues with the content",
		},
		"strengths": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "List of content strengths and positive aspects",
		},
		"recommendations": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "Specific recommendations for improvement",
		},
	},
	"required": []interface{}{"score", "issues"},
}

// seoMetadataSchema defines the expected structure for SEO metadata output
var seoMetadataSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"title": map[string]interface{}{
			"type":        "string",
			"maxLength":   60,
			"description": "SEO-optimized title (max 60 characters)",
		},
		"meta_description": map[string]interface{}{
			"type":        "string",
			"maxLength":   160,
			"description": "Meta description for search engines (max 160 characters)",
		},
		"keywords": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"maxItems":    10,
			"description": "List of relevant SEO keywords (max 10)",
		},
		"slug": map[string]interface{}{
			"type":        "string",
			"pattern":     "^[a-z0-9-]+$",
			"description": "URL-friendly slug (lowercase, numbers, hyphens only)",
		},
	},
	"required": []interface{}{"title", "meta_description", "keywords", "slug"},
}

// getQualityAnalysisSchemaJSON returns the quality analysis schema as JSON string
func getQualityAnalysisSchemaJSON() string {
	schemaBytes, err := json.Marshal(qualityAnalysisSchema)
	if err != nil {
		return "{}"
	}
	return string(schemaBytes)
}

// getSEOMetadataSchemaJSON returns the SEO metadata schema as JSON string
func getSEOMetadataSchemaJSON() string {
	schemaBytes, err := json.Marshal(seoMetadataSchema)
	if err != nil {
		return "{}"
	}
	return string(schemaBytes)
}

// validateQualityAnalysis validates quality analysis output against schema
func validateQualityAnalysis(data map[string]interface{}) error {
	score, ok := data["score"]
	if !ok {
		return fmt.Errorf("missing required field: score")
	}

	scoreFloat, ok := score.(float64)
	if !ok {
		return fmt.Errorf("score must be a number")
	}

	if scoreFloat < 0 || scoreFloat > 100 {
		return fmt.Errorf("score must be between 0 and 100")
	}

	if _, ok := data["issues"]; !ok {
		return fmt.Errorf("missing required field: issues")
	}

	return nil
}

// validateSEOMetadata validates SEO metadata output against schema
func validateSEOMetadata(data map[string]interface{}) error {
	requiredFields := []string{"title", "meta_description", "keywords", "slug"}

	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate title length
	if title, ok := data["title"].(string); ok {
		if len(title) > 60 {
			return fmt.Errorf("title exceeds 60 characters")
		}
	}

	// Validate meta_description length
	if metaDesc, ok := data["meta_description"].(string); ok {
		if len(metaDesc) > 160 {
			return fmt.Errorf("meta_description exceeds 160 characters")
		}
	}

	return nil
}

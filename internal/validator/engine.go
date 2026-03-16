package validator

import (
	"embed"
	"fmt"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/*.json
var schemaFiles embed.FS

var (
	// We use a sync.Map to store compiled schemas for thread-safe concurrent access
	// sync.Map ensures that if your Lambda only ever receives fintech events,
	// it never wastes memory loading the healthcare schema.
	// map[string]*gojsonschema.Schema
	schemaCache sync.Map
)

// ValidateEvent selects the correct schema based on schemaType and validates the JSON
func ValidateEvent(schemaType string, jsonString string) error {
	// 1. Guard Clause: Fail fast if the type isn't in our registry
	if !IsValidSchema(schemaType) {
		return fmt.Errorf("unsupported schema_type: %s", schemaType)
	}

	validator, err := getValidator(schemaType)
	if err != nil {
		return err
	}

	documentLoader := gojsonschema.NewStringLoader(jsonString)
	result, err := validator.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation execution error: %w", err)
	}

	if !result.Valid() {
		var report string
		for _, desc := range result.Errors() {
			report += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf("schema validation failed for %s:\n%s", schemaType, report)
	}

	return nil
}

// getValidator retrieves a compiled schema from cache or compiles it if missing
func getValidator(schemaType string) (*gojsonschema.Schema, error) {
	// 1. Check if we already have this schema compiled in memory
	if val, ok := schemaCache.Load(schemaType); ok {
		return val.(*gojsonschema.Schema), nil
	}

	// 2. If not, find the file in our embedded filesystem
	fileName := fmt.Sprintf("schemas/%s.json", schemaType)
	rawSchema, err := schemaFiles.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("schema type '%s' not supported (file %s not found)", schemaType, fileName)
	}

	// 3. Compile the schema (CPU intensive)
	loader := gojsonschema.NewBytesLoader(rawSchema)
	compiled, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema %s: %w", schemaType, err)
	}

	// 4. Store in cache for future Warm Starts
	schemaCache.Store(schemaType, compiled)

	return compiled, nil
}

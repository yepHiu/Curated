package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Schema is a minimal JSON Schema subset used to validate tool arguments.
// additionalProperties defaults to false.
type Schema struct {
	Type                 string
	Description          string
	Properties           map[string]Schema
	Required             []string
	AdditionalProperties *bool
	Enum                 []string
	Minimum              *float64
	Maximum              *float64
	MinLength            int
	MaxLength            int
	MinItems             int
	MaxItems             int
	Items                *Schema
}

func (s Schema) additionalAllowed() bool {
	return s.AdditionalProperties != nil && *s.AdditionalProperties
}

// ValidateArgs unmarshals raw JSON and rejects unknown fields / wrong types.
func ValidateArgs(schema Schema, raw json.RawMessage) (map[string]any, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		trimmed = "{}"
	}
	var value any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return nil, fmt.Errorf("arguments are not valid JSON")
	}
	if err := validateValue(schema, value, ""); err != nil {
		return nil, err
	}
	obj, _ := value.(map[string]any)
	if obj == nil {
		obj = map[string]any{}
	}
	return obj, nil
}

func validateValue(schema Schema, value any, path string) error {
	typ := strings.TrimSpace(schema.Type)
	if typ == "" {
		typ = "object"
	}
	loc := path
	if loc == "" {
		loc = "$"
	}
	switch typ {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected object", loc)
		}
		if !schema.additionalAllowed() {
			for key := range obj {
				if _, known := schema.Properties[key]; !known {
					return fmt.Errorf("%s: unknown field %q", loc, key)
				}
			}
		}
		for _, req := range schema.Required {
			if _, ok := obj[req]; !ok {
				return fmt.Errorf("%s: missing required field %q", loc, req)
			}
		}
		for key, prop := range schema.Properties {
			child, exists := obj[key]
			if !exists {
				continue
			}
			if err := validateValue(prop, child, loc+"."+key); err != nil {
				return err
			}
		}
		return nil
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s: expected string", loc)
		}
		if schema.MinLength > 0 && len([]rune(str)) < schema.MinLength {
			return fmt.Errorf("%s: shorter than %d", loc, schema.MinLength)
		}
		if schema.MaxLength > 0 && len([]rune(str)) > schema.MaxLength {
			return fmt.Errorf("%s: longer than %d", loc, schema.MaxLength)
		}
		if len(schema.Enum) > 0 && !containsString(schema.Enum, str) {
			return fmt.Errorf("%s: value %q is not allowed", loc, str)
		}
		return nil
	case "integer":
		n, ok := jsonNumber(value)
		if !ok || n != float64(int64(n)) {
			return fmt.Errorf("%s: expected integer", loc)
		}
		return checkRange(schema, n, loc)
	case "number":
		n, ok := jsonNumber(value)
		if !ok {
			return fmt.Errorf("%s: expected number", loc)
		}
		return checkRange(schema, n, loc)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: expected boolean", loc)
		}
		return nil
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s: expected array", loc)
		}
		if schema.MinItems > 0 && len(arr) < schema.MinItems {
			return fmt.Errorf("%s: fewer than %d items", loc, schema.MinItems)
		}
		if schema.MaxItems > 0 && len(arr) > schema.MaxItems {
			return fmt.Errorf("%s: more than %d items", loc, schema.MaxItems)
		}
		if schema.Items == nil {
			return nil
		}
		for i, item := range arr {
			if err := validateValue(*schema.Items, item, fmt.Sprintf("%s[%d]", loc, i)); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: unsupported schema type %q", loc, typ)
	}
}

func checkRange(schema Schema, n float64, loc string) error {
	if schema.Minimum != nil && n < *schema.Minimum {
		return fmt.Errorf("%s: below minimum %v", loc, *schema.Minimum)
	}
	if schema.Maximum != nil && n > *schema.Maximum {
		return fmt.Errorf("%s: above maximum %v", loc, *schema.Maximum)
	}
	return nil
}

func jsonNumber(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		return n, true
	case json.Number:
		v, err := n.Float64()
		return v, err == nil
	default:
		return 0, false
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// JSONSchemaMap renders the schema as an OpenAI-compatible parameters object.
func (s Schema) JSONSchemaMap() map[string]any {
	typ := strings.TrimSpace(s.Type)
	if typ == "" {
		typ = "object"
	}
	out := map[string]any{"type": typ}
	if s.Description != "" {
		out["description"] = s.Description
	}
	if typ == "object" {
		props := map[string]any{}
		for name, child := range s.Properties {
			props[name] = child.JSONSchemaMap()
		}
		out["properties"] = props
		if len(s.Required) > 0 {
			out["required"] = s.Required
		}
		out["additionalProperties"] = s.additionalAllowed()
	}
	if s.Items != nil {
		out["items"] = s.Items.JSONSchemaMap()
	}
	if len(s.Enum) > 0 {
		out["enum"] = s.Enum
	}
	if s.Minimum != nil {
		out["minimum"] = *s.Minimum
	}
	if s.Maximum != nil {
		out["maximum"] = *s.Maximum
	}
	if s.MinLength > 0 {
		out["minLength"] = s.MinLength
	}
	if s.MaxLength > 0 {
		out["maxLength"] = s.MaxLength
	}
	if s.MinItems > 0 {
		out["minItems"] = s.MinItems
	}
	if s.MaxItems > 0 {
		out["maxItems"] = s.MaxItems
	}
	return out
}

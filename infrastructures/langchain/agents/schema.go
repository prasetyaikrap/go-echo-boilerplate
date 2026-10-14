package agents

import (
	"reflect"
	"strings"
	"time"
)

// schemaFor builds the JSON schema of a tool input struct from the same tags as HTTP request models:
//   - json:"name"            field name (json:"-" skips the field)
//   - validate:"required"    listed in "required"; validate:"oneof=a b" becomes an enum
//   - description:"..."      tells the model what to put in the field
func schemaFor(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t == reflect.TypeFor[time.Time]() {
		return map[string]any{"type": "string", "format": "date-time"}
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaFor(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(t.Elem())}
	case reflect.Struct:
		return structSchema(t)
	default:
		return map[string]any{}
	}
}

func structSchema(t reflect.Type) map[string]any {
	properties := map[string]any{}
	var required []string

	var addFields func(t reflect.Type)
	addFields = func(t reflect.Type) {
		for field := range t.Fields() {
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}

			// Embedded structs without a json name are flattened like encoding/json does, even when the embedded
			// type itself is unexported (its exported fields are still decoded)
			if field.Anonymous && name == "" && field.Type.Kind() == reflect.Struct {
				addFields(field.Type)
				continue
			}

			if !field.IsExported() {
				continue
			}
			if name == "" {
				name = field.Name
			}

			property := schemaFor(field.Type)
			if description := field.Tag.Get("description"); description != "" {
				property["description"] = description
			}

			for rule := range strings.SplitSeq(field.Tag.Get("validate"), ",") {
				switch {
				case rule == "required":
					required = append(required, name)
				case strings.HasPrefix(rule, "oneof="):
					property["enum"] = strings.Fields(strings.TrimPrefix(rule, "oneof="))
				}
			}

			properties[name] = property
		}
	}
	addFields(t)

	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

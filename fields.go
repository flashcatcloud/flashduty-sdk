package flashduty

import (
	"context"
)

// ListFieldsInput contains parameters for listing custom fields
type ListFieldsInput struct {
	FieldIDs  []string // Filter by field IDs
	FieldName string   // Filter by exact field name
}

// ListFieldsOutput contains the result of listing fields
type ListFieldsOutput struct {
	Fields []FieldInfo `json:"fields"`
	Total  int         `json:"total"`
}

// ListFields queries custom field definitions
func (c *Client) ListFields(ctx context.Context, input *ListFieldsInput) (*ListFieldsOutput, error) {
	result, err := postData[struct {
		Items []struct {
			FieldID      string   `json:"field_id"`
			FieldName    string   `json:"field_name"`
			DisplayName  string   `json:"display_name"`
			FieldType    string   `json:"field_type"`
			ValueType    string   `json:"value_type"`
			Options      []string `json:"options,omitempty"`
			DefaultValue any      `json:"default_value,omitempty"`
		} `json:"items"`
	}](c, ctx, "/field/list", map[string]any{}, "failed to list fields")
	if err != nil {
		return nil, err
	}

	// Build filter ID set
	filterIDSet := make(map[string]struct{})
	for _, id := range input.FieldIDs {
		filterIDSet[id] = struct{}{}
	}

	fields := []FieldInfo{}
	for _, f := range result.Items {
		// Filter by ID if provided
		if len(filterIDSet) > 0 {
			if _, ok := filterIDSet[f.FieldID]; !ok {
				continue
			}
		}

		// Filter by name if provided
		if input.FieldName != "" && f.FieldName != input.FieldName {
			continue
		}

		fields = append(fields, FieldInfo{
			FieldID:      f.FieldID,
			FieldName:    f.FieldName,
			DisplayName:  f.DisplayName,
			FieldType:    f.FieldType,
			ValueType:    f.ValueType,
			Options:      f.Options,
			DefaultValue: f.DefaultValue,
		})
	}

	return &ListFieldsOutput{
		Fields: fields,
		Total:  len(fields),
	}, nil
}

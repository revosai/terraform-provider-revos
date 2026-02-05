package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJsonEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical JSON",
			a:        `{"foo": "bar"}`,
			b:        `{"foo": "bar"}`,
			expected: true,
		},
		{
			name:     "different key order",
			a:        `{"a": 1, "b": 2}`,
			b:        `{"b": 2, "a": 1}`,
			expected: true,
		},
		{
			name:     "nested objects different order",
			a:        `{"joins":{"clerk_Organizations":{"relationship":"one_to_many","sql":"x"},"clerk_Users":{"relationship":"one_to_many","sql":"y"}}}`,
			b:        `{"joins":{"clerk_Users":{"sql":"y","relationship":"one_to_many"},"clerk_Organizations":{"sql":"x","relationship":"one_to_many"}}}`,
			expected: true,
		},
		{
			name:     "different values",
			a:        `{"foo": "bar"}`,
			b:        `{"foo": "baz"}`,
			expected: false,
		},
		{
			name:     "different keys",
			a:        `{"foo": "bar"}`,
			b:        `{"baz": "bar"}`,
			expected: false,
		},
		{
			name:     "extra key",
			a:        `{"foo": "bar"}`,
			b:        `{"foo": "bar", "extra": "value"}`,
			expected: false,
		},
		{
			name:     "arrays same order",
			a:        `{"arr": [1, 2, 3]}`,
			b:        `{"arr": [1, 2, 3]}`,
			expected: true,
		},
		{
			name:     "arrays different order",
			a:        `{"arr": [1, 2, 3]}`,
			b:        `{"arr": [3, 2, 1]}`,
			expected: false,
		},
		{
			name:     "nested arrays",
			a:        `{"arr": [{"a": 1}, {"b": 2}]}`,
			b:        `{"arr": [{"a": 1}, {"b": 2}]}`,
			expected: true,
		},
		{
			name:     "invalid JSON a",
			a:        `not json`,
			b:        `{"foo": "bar"}`,
			expected: false,
		},
		{
			name:     "invalid JSON b",
			a:        `{"foo": "bar"}`,
			b:        `not json`,
			expected: false,
		},
		{
			name:     "empty objects",
			a:        `{}`,
			b:        `{}`,
			expected: true,
		},
		{
			name:     "null values",
			a:        `{"foo": null}`,
			b:        `{"foo": null}`,
			expected: true,
		},
		{
			name:     "number types",
			a:        `{"num": 42}`,
			b:        `{"num": 42}`,
			expected: true,
		},
		{
			name:     "boolean values",
			a:        `{"flag": true}`,
			b:        `{"flag": true}`,
			expected: true,
		},
		{
			name:     "different boolean values",
			a:        `{"flag": true}`,
			b:        `{"flag": false}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("jsonEqual(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestDeepEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		{
			name:     "equal strings",
			a:        "foo",
			b:        "foo",
			expected: true,
		},
		{
			name:     "different strings",
			a:        "foo",
			b:        "bar",
			expected: false,
		},
		{
			name:     "equal numbers",
			a:        float64(42),
			b:        float64(42),
			expected: true,
		},
		{
			name:     "different numbers",
			a:        float64(42),
			b:        float64(43),
			expected: false,
		},
		{
			name:     "equal booleans",
			a:        true,
			b:        true,
			expected: true,
		},
		{
			name:     "different booleans",
			a:        true,
			b:        false,
			expected: false,
		},
		{
			name:     "equal nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "nil vs value",
			a:        nil,
			b:        "foo",
			expected: false,
		},
		{
			name: "equal maps",
			a:    map[string]interface{}{"a": "b"},
			b:    map[string]interface{}{"a": "b"},
			expected: true,
		},
		{
			name: "maps different values",
			a:    map[string]interface{}{"a": "b"},
			b:    map[string]interface{}{"a": "c"},
			expected: false,
		},
		{
			name: "maps different keys",
			a:    map[string]interface{}{"a": "b"},
			b:    map[string]interface{}{"x": "b"},
			expected: false,
		},
		{
			name: "maps different lengths",
			a:    map[string]interface{}{"a": "b"},
			b:    map[string]interface{}{"a": "b", "c": "d"},
			expected: false,
		},
		{
			name:     "equal slices",
			a:        []interface{}{"a", "b"},
			b:        []interface{}{"a", "b"},
			expected: true,
		},
		{
			name:     "slices different order",
			a:        []interface{}{"a", "b"},
			b:        []interface{}{"b", "a"},
			expected: false,
		},
		{
			name:     "slices different lengths",
			a:        []interface{}{"a"},
			b:        []interface{}{"a", "b"},
			expected: false,
		},
		{
			name:     "map vs slice",
			a:        map[string]interface{}{"a": "b"},
			b:        []interface{}{"a", "b"},
			expected: false,
		},
		{
			name:     "nested structures equal",
			a:        map[string]interface{}{"nested": map[string]interface{}{"deep": []interface{}{1, 2}}},
			b:        map[string]interface{}{"nested": map[string]interface{}{"deep": []interface{}{1, 2}}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("deepEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestStringEqualOrBothEmpty(t *testing.T) {
	tests := []struct {
		name     string
		a        types.String
		b        types.String
		expected bool
	}{
		{
			name:     "both null",
			a:        types.StringNull(),
			b:        types.StringNull(),
			expected: true,
		},
		{
			name:     "both empty string",
			a:        types.StringValue(""),
			b:        types.StringValue(""),
			expected: true,
		},
		{
			name:     "null and empty string",
			a:        types.StringNull(),
			b:        types.StringValue(""),
			expected: true,
		},
		{
			name:     "empty string and null",
			a:        types.StringValue(""),
			b:        types.StringNull(),
			expected: true,
		},
		{
			name:     "equal non-empty",
			a:        types.StringValue("foo"),
			b:        types.StringValue("foo"),
			expected: true,
		},
		{
			name:     "different non-empty",
			a:        types.StringValue("foo"),
			b:        types.StringValue("bar"),
			expected: false,
		},
		{
			name:     "null and non-empty",
			a:        types.StringNull(),
			b:        types.StringValue("foo"),
			expected: false,
		},
		{
			name:     "non-empty and null",
			a:        types.StringValue("foo"),
			b:        types.StringNull(),
			expected: false,
		},
		{
			name:     "empty and non-empty",
			a:        types.StringValue(""),
			b:        types.StringValue("foo"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringEqualOrBothEmpty(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("stringEqualOrBothEmpty(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestJsonSemanticEqualModifier_PlanModifyString(t *testing.T) {
	ctx := context.Background()
	modifier := jsonSemanticEqualModifier{}

	tests := []struct {
		name          string
		stateValue    types.String
		configValue   types.String
		expectedPlan  types.String
		expectChanged bool
	}{
		{
			name:          "null state - no comparison",
			stateValue:    types.StringNull(),
			configValue:   types.StringValue(`{"a": 1}`),
			expectedPlan:  types.StringValue(`{"a": 1}`),
			expectChanged: false,
		},
		{
			name:          "null config - no change",
			stateValue:    types.StringValue(`{"a": 1}`),
			configValue:   types.StringNull(),
			expectedPlan:  types.StringNull(),
			expectChanged: false,
		},
		{
			name:          "unknown config - no change",
			stateValue:    types.StringValue(`{"a": 1}`),
			configValue:   types.StringUnknown(),
			expectedPlan:  types.StringUnknown(),
			expectChanged: false,
		},
		{
			name:          "semantically equal - use state",
			stateValue:    types.StringValue(`{"a": 1, "b": 2}`),
			configValue:   types.StringValue(`{"b": 2, "a": 1}`),
			expectedPlan:  types.StringValue(`{"a": 1, "b": 2}`),
			expectChanged: true,
		},
		{
			name:          "semantically different - keep config",
			stateValue:    types.StringValue(`{"a": 1}`),
			configValue:   types.StringValue(`{"a": 2}`),
			expectedPlan:  types.StringValue(`{"a": 2}`),
			expectChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				StateValue:  tt.stateValue,
				ConfigValue: tt.configValue,
				PlanValue:   tt.configValue, // Plan starts as config value
			}
			resp := &planmodifier.StringResponse{
				PlanValue: tt.configValue,
			}

			modifier.PlanModifyString(ctx, req, resp)

			if tt.expectChanged {
				if !resp.PlanValue.Equal(tt.expectedPlan) {
					t.Errorf("PlanValue = %v, want %v", resp.PlanValue, tt.expectedPlan)
				}
			} else {
				// If not changed, plan should still be config value
				if !resp.PlanValue.Equal(tt.configValue) {
					t.Errorf("PlanValue should not have changed, got %v, want %v", resp.PlanValue, tt.configValue)
				}
			}
		})
	}
}

func TestJsonSemanticEqualModifier_Description(t *testing.T) {
	ctx := context.Background()
	modifier := jsonSemanticEqualModifier{}

	desc := modifier.Description(ctx)
	if desc == "" {
		t.Error("Description should not be empty")
	}

	mdDesc := modifier.MarkdownDescription(ctx)
	if mdDesc != desc {
		t.Errorf("MarkdownDescription = %q, want %q", mdDesc, desc)
	}
}

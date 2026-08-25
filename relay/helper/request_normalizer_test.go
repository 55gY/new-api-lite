package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanNullValuesFiltersConsecutiveArrayNulls(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"required": nil,
		"items": []any{
			nil,
			nil,
			map[string]any{
				"name":   "tool",
				"unused": nil,
			},
			nil,
			"kept",
			nil,
		},
		"empty": []any{nil, nil},
	}

	cleanNullValues(schema)

	require.NotContains(t, schema, "required")
	require.Equal(t, []any{}, schema["empty"])

	items, ok := schema["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	require.Equal(t, "kept", items[1])

	nested, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tool", nested["name"])
	require.NotContains(t, nested, "unused")
}

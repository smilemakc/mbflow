package builtin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVToJSONExecutor_Execute(t *testing.T) {
	executor := NewCSVToJSONExecutor()
	ctx := context.Background()

	t.Run("basic CSV with headers", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := "name,age,city\nJohn,30,NYC\nJane,25,LA"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		assert.True(t, result["success"].(bool))
		assert.Equal(t, 2, result["row_count"].(int))
		assert.Equal(t, 3, result["column_count"].(int))

		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 2)
		assert.Equal(t, "John", rows[0]["name"])
		assert.Equal(t, "30", rows[0]["age"])
		assert.Equal(t, "NYC", rows[0]["city"])
		assert.Equal(t, "Jane", rows[1]["name"])
		assert.Equal(t, "25", rows[1]["age"])
		assert.Equal(t, "LA", rows[1]["city"])
	})

	t.Run("CSV without headers - auto-generated", func(t *testing.T) {
		config := map[string]any{
			"has_header": false,
		}
		input := "John,30,NYC\nJane,25,LA"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 2)
		assert.Equal(t, "John", rows[0]["col_0"])
		assert.Equal(t, "30", rows[0]["col_1"])
		assert.Equal(t, "NYC", rows[0]["col_2"])
	})

	t.Run("CSV with custom headers", func(t *testing.T) {
		config := map[string]any{
			"has_header":     false,
			"custom_headers": []any{"first_name", "years", "location"},
		}
		input := "John,30,NYC\nJane,25,LA"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Equal(t, "John", rows[0]["first_name"])
		assert.Equal(t, "30", rows[0]["years"])
		assert.Equal(t, "NYC", rows[0]["location"])
	})

	t.Run("semicolon delimiter", func(t *testing.T) {
		config := map[string]any{
			"delimiter":  ";",
			"has_header": true,
		}
		input := "name;age;city\nJohn;30;NYC\nJane;25;LA"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 2)
		assert.Equal(t, "John", rows[0]["name"])
	})

	t.Run("tab delimiter", func(t *testing.T) {
		config := map[string]any{
			"delimiter":  "\t",
			"has_header": true,
		}
		input := "name\tage\tcity\nJohn\t30\tNYC"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 1)
		assert.Equal(t, "John", rows[0]["name"])
	})

	t.Run("pipe delimiter", func(t *testing.T) {
		config := map[string]any{
			"delimiter":  "|",
			"has_header": true,
		}
		input := "name|age|city\nJohn|30|NYC"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Equal(t, "John", rows[0]["name"])
	})

	t.Run("trim spaces", func(t *testing.T) {
		config := map[string]any{
			"has_header":  true,
			"trim_spaces": true,
		}
		input := "  name  ,  age  ,  city  \n  John  ,  30  ,  NYC  "

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		headers := result["headers"].([]string)
		assert.Equal(t, "name", headers[0])
		assert.Equal(t, "age", headers[1])

		rows := result["result"].([]map[string]any)
		assert.Equal(t, "John", rows[0]["name"])
		assert.Equal(t, "30", rows[0]["age"])
	})

	t.Run("skip empty rows", func(t *testing.T) {
		config := map[string]any{
			"has_header":      true,
			"skip_empty_rows": true,
		}
		input := "name,age\nJohn,30\n,,\nJane,25\n  ,  "

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 2)
		assert.Equal(t, "John", rows[0]["name"])
		assert.Equal(t, "Jane", rows[1]["name"])
	})

	t.Run("keep empty rows when disabled", func(t *testing.T) {
		config := map[string]any{
			"has_header":      true,
			"skip_empty_rows": false,
		}
		input := "name,age\nJohn,30\n,\nJane,25"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 3)
	})

	t.Run("quoted values", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := `name,bio,age
"John ""Johnny"" Doe","Hello, World!",30
Jane,"Line1
Line2",25`

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 2)
		assert.Equal(t, `John "Johnny" Doe`, rows[0]["name"])
		assert.Equal(t, "Hello, World!", rows[0]["bio"])
		assert.Contains(t, rows[1]["bio"], "Line1")
	})

	t.Run("input from map with auto-detect", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := map[string]any{
			"csv":  "name,age\nJohn,30",
			"meta": "some metadata",
		}

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Equal(t, "John", rows[0]["name"])
	})

	t.Run("input from map with custom key", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
			"input_key":  "csv_data",
		}
		input := map[string]any{
			"csv_data": "name,age\nJane,25",
			"meta":     "some metadata",
		}

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Equal(t, "Jane", rows[0]["name"])
	})

	t.Run("input as bytes", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := []byte("name,age\nJohn,30")

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Equal(t, "John", rows[0]["name"])
	})

	t.Run("empty CSV", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := ""

		_, err := executor.Execute(ctx, config, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("headers only", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := "name,age,city"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		assert.Equal(t, 0, result["row_count"].(int))
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 0)
	})

	t.Run("uneven columns", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}
		input := "name,age,city\nJohn,30"

		output, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		result := output.(map[string]any)
		rows := result["result"].([]map[string]any)
		assert.Len(t, rows, 1)
		assert.Equal(t, "John", rows[0]["name"])
		assert.Equal(t, "30", rows[0]["age"])
		// city should not be present since the row has fewer columns
		_, exists := rows[0]["city"]
		assert.False(t, exists)
	})

	t.Run("performance with many rows", func(t *testing.T) {
		config := map[string]any{
			"has_header": true,
		}

		// Generate CSV with 1000 rows
		var sb strings.Builder
		sb.WriteString("id,name,value\n")
		for i := 0; i < 1000; i++ {
			sb.WriteString(fmt.Sprintf("%d,name%d,value%d\n", i, i, i))
		}

		output, err := executor.Execute(ctx, config, sb.String())
		require.NoError(t, err)

		result := output.(map[string]any)
		assert.Equal(t, 1000, result["row_count"].(int))
		assert.True(t, result["duration_ms"].(int64) < 1000) // Should complete in under 1 second
	})
}

func TestCSVToJSONExecutor_Validate(t *testing.T) {
	executor := NewCSVToJSONExecutor()

	t.Run("valid config", func(t *testing.T) {
		config := map[string]any{
			"delimiter":  ",",
			"has_header": true,
		}
		err := executor.Validate(config)
		assert.NoError(t, err)
	})

	t.Run("valid tab delimiter", func(t *testing.T) {
		config := map[string]any{
			"delimiter": "\\t",
		}
		err := executor.Validate(config)
		assert.NoError(t, err)
	})

	t.Run("empty delimiter", func(t *testing.T) {
		config := map[string]any{
			"delimiter": "",
		}
		err := executor.Validate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delimiter")
	})

	t.Run("invalid delimiter - too long", func(t *testing.T) {
		config := map[string]any{
			"delimiter": ",,",
		}
		err := executor.Validate(config)
		assert.Error(t, err)
	})

	t.Run("default config", func(t *testing.T) {
		config := map[string]any{}
		err := executor.Validate(config)
		assert.NoError(t, err)
	})
}

func TestCSVToJSONExecutor_NodeType(t *testing.T) {
	executor := NewCSVToJSONExecutor()
	assert.Equal(t, "csv_to_json", executor.NodeType)
}

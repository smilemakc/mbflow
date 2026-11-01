package builtin

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gojq normalizes numbers in place; the executor must not mutate its input,
// which is another node's output shared with observers.
func TestTransformJQ_DoesNotMutateInput(t *testing.T) {
	nested := map[string]any{"n": int64(7), "list": []any{int32(1), map[string]any{"x": uint8(2)}}}
	input := map[string]any{"items": []map[string]any{{"v": int64(3)}}, "nested": nested}

	out, err := NewTransformExecutor().Execute(context.Background(),
		map[string]any{"type": "jq", "filter": ".nested.n + .items[0].v"}, input)
	require.NoError(t, err)
	assert.EqualValues(t, 10, out)

	assert.IsType(t, int64(0), nested["n"], "input map must not be rewritten")
	assert.IsType(t, int32(0), nested["list"].([]any)[0])
	assert.IsType(t, uint8(0), nested["list"].([]any)[1].(map[string]any)["x"])
	assert.IsType(t, int64(0), input["items"].([]map[string]any)[0]["v"])
}

// Run with -race: serializing the input while jq runs must not race.
func TestTransformJQ_ConcurrentReadersOfInput(t *testing.T) {
	input := map[string]any{"body": []any{map[string]any{"a": int64(1)}, map[string]any{"a": int64(2)}}}
	exec := NewTransformExecutor()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := exec.Execute(context.Background(), map[string]any{"type": "jq", "filter": "[.body[].a] | add"}, input)
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			_, err := json.Marshal(input)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestCloneJQInput_Scalars(t *testing.T) {
	assert.Equal(t, "s", cloneJQInput("s"))
	assert.Nil(t, cloneJQInput(nil))
}

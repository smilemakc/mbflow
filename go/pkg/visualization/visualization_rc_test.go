package visualization

import (
	"strings"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rcDiamond() *models.Workflow {
	return &models.Workflow{
		Name: "diamond",
		Nodes: []*models.Node{
			{ID: "a", Name: "A", Type: "http", Config: map[string]any{"url": "https://x"}},
			{ID: "b", Name: "B", Type: "transform"},
			{ID: "c", Name: "C", Type: "transform"},
			{ID: "d", Name: "D", Type: "merge"},
		},
		Edges: []*models.Edge{
			{ID: "1", From: "a", To: "b"},
			{ID: "2", From: "a", To: "c"},
			{ID: "3", From: "b", To: "d"},
			{ID: "4", From: "c", To: "d"},
		},
	}
}

func TestRCASCII_DiamondIsNotACycle(t *testing.T) {
	out, err := NewASCIIRenderer().Render(rcDiamond(), &RenderOptions{ShowConfig: true})
	require.NoError(t, err)
	assert.NotContains(t, out, "cycle detected")
	assert.Equal(t, 1, strings.Count(out, "[d] D (merge)"), out)
	assert.Contains(t, out, "[d] (see above)")
	assert.Contains(t, out, "│ https://x")
}

func TestRCASCII_RealCycle(t *testing.T) {
	wf := &models.Workflow{
		ID: "wf-cycle",
		Nodes: []*models.Node{
			{ID: "a", Type: "transform"},
			{ID: "b", Type: "transform"},
		},
		Edges: []*models.Edge{
			{ID: "1", From: "a", To: "b"},
			{ID: "2", From: "b", To: "a"},
		},
	}
	out, err := NewASCIIRenderer().Render(wf, &RenderOptions{CompactMode: true})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "wf-cycle\n"))
	assert.Contains(t, out, "(cycle detected: a)")
}

func TestRCASCII_DoesNotMutateOptions(t *testing.T) {
	opts := &RenderOptions{UseColor: true}
	_, err := NewASCIIRenderer().Render(rcDiamond(), opts)
	require.NoError(t, err)
	assert.True(t, opts.UseColor, "caller options must not be modified")
}

func TestRCASCII_NonLastSharedChildAndColor(t *testing.T) {
	r := NewASCIIRenderer()
	// Shared child rendered under a non-last branch exercises the "├──" prefix.
	wf := &models.Workflow{
		Name: "w",
		Nodes: []*models.Node{
			{ID: "a", Type: "t"}, {ID: "b", Type: "t"}, {ID: "c", Type: "t"},
		},
		Edges: []*models.Edge{
			{ID: "1", From: "a", To: "b"},
			{ID: "2", From: "a", To: "c"},
			{ID: "3", From: "b", To: "c"},
		},
	}
	out, err := r.Render(wf, &RenderOptions{})
	require.NoError(t, err)
	assert.Contains(t, out, "[c] (see above)")
	assert.Equal(t, "\033[31mx"+colorReset, r.colorize("x", "\033[31m", true))
}

func TestRCMermaid_Deterministic(t *testing.T) {
	wf := rcDiamond()
	opts := DefaultRenderOptions()
	opts.ThemeVariables = map[string]string{"primaryColor": "#fff", "lineColor": "#000", "accent": "#123"}
	first, err := NewMermaidRenderer().Render(wf, opts)
	require.NoError(t, err)
	for i := 0; i < 20; i++ {
		again, err := NewMermaidRenderer().Render(wf, opts)
		require.NoError(t, err)
		require.Equal(t, first, again)
	}
	assert.Less(t, strings.Index(first, "accent"), strings.Index(first, "lineColor"))
	assert.Less(t, strings.Index(first, "a --> b & c"), strings.Index(first, "b --> d"))
	assert.Less(t, strings.Index(first, "class a httpNode"), strings.Index(first, "class d mergeNode"))
}

func TestRCMermaid_ConfigAndEdges(t *testing.T) {
	wf := &models.Workflow{
		Name: "x",
		Nodes: []*models.Node{
			{ID: "h", Type: "http", Config: map[string]any{"method": "POST", "url": "https://api"}},
			{ID: "h2", Type: "http", Config: map[string]any{"x": 1}},
			{ID: "l1", Type: "llm", Config: map[string]any{"provider": "openai", "model": "gpt-4o"}},
			{ID: "l2", Type: "llm", Config: map[string]any{"provider": "anthropic"}},
			{ID: "l3", Type: "llm", Config: map[string]any{"provider": "gemini"}},
			{ID: "l4", Type: "llm", Config: map[string]any{"temperature": 1}},
			{ID: "r", Name: `Say "hi"`, Type: "llm", Config: map[string]any{
				"provider": "openai-responses", "model": "gpt-5",
				"hosted_tools": []any{
					map[string]any{"type": "web_search_preview"},
					map[string]any{"type": "file_search"},
					map[string]any{"type": "code_interpreter"},
					map[string]any{"type": "mcp"},
					map[string]any{"no_type": true},
					"not a map",
				},
				"reasoning": map[string]any{"effort": "high"},
			}},
			{ID: "r2", Type: "llm", Config: map[string]any{"provider": "openai-responses", "hosted_tools": []any{"x"}}},
			{ID: "t", Type: "transform", Config: map[string]any{"type": "jq"}},
			{ID: "t2", Type: "transform", Config: map[string]any{"other": 1}},
			{ID: "if", Type: "conditional", Config: map[string]any{"a": 1}},
			{ID: "m", Type: "merge"},
			{ID: "custom", Type: "telegram", Config: map[string]any{"a": 1}},
		},
		Edges: []*models.Edge{
			{ID: "1", From: "if", To: "h", Condition: `x < 1 && y > "2"`},
			{ID: "2", From: "if", To: "m"},
			{ID: "3", From: "m", To: "if", Loop: &models.LoopConfig{MaxIterations: 3}},
			{ID: "4", From: "h", To: "t", Condition: "ok"},
		},
	}
	out, err := NewMermaidRenderer().Render(wf, &RenderOptions{ShowConfig: true, ShowConditions: true, Direction: "elk"})
	require.NoError(t, err)

	for _, want := range []string{
		"layout: elk",
		"flowchart TB",
		`h["HTTP: POST: h<br/>https://api"]`,
		`h2["HTTP: h2"]`,
		`l1(["OpenAI: l1<br/>gpt-4o"])`,
		`l2(["Anthropic: l2"])`,
		`l3(["GEMINI: l3"])`,
		`l4(["LLM: l4"])`,
		"OpenAI Responses: Say &quot;hi&quot;",
		"🌐 Web Search, 📄 File Search, 💻 Code, mcp",
		"💭 high reasoning",
		`r2(["OpenAI Responses: r2"])`,
		`t[/"Transform: t<br/>jq"/]`,
		`t2[/"Transform: t2"/]`,
		`if{"If: if"}`,
		`m{{"Merge: m"}}`,
		`custom["TELEGRAM: custom"]`,
		`if -- "x &lt; 1 &amp;&amp; y &gt; &quot;2&quot;" --> h`,
		"if --> m",
		`m -. "loop (max 3)" .-> if`,
		`h -- "ok" --> t`,
		"classDef httpNode",
	} {
		assert.Contains(t, out, want)
	}
	assert.NotContains(t, out, "class custom")

	// Without conditions shown, multiple plain edges collapse with "&".
	plain, err := NewMermaidRenderer().Render(&models.Workflow{
		Nodes: []*models.Node{{ID: "a", Type: "x"}, {ID: "b", Type: "x"}, {ID: "c", Type: "x"}},
		Edges: []*models.Edge{{ID: "1", From: "a", To: "b", Condition: "c1"}, {ID: "2", From: "a", To: "c"}},
	}, &RenderOptions{Direction: "LR"})
	require.NoError(t, err)
	assert.Contains(t, plain, "flowchart LR")
	assert.Contains(t, plain, "a --> b & c")
	assert.NotContains(t, plain, "---\nconfig")

	_, err = NewMermaidRenderer().Render(nil, nil)
	assert.Error(t, err)
	def, err := NewMermaidRenderer().Render(&models.Workflow{Nodes: []*models.Node{{ID: "a", Type: "x"}}}, nil)
	require.NoError(t, err)
	assert.Contains(t, def, "flowchart TB")
}

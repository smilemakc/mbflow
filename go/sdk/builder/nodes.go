package builder

import "github.com/smilemakc/mbflow/go/sdk/models"

// --- HTTP Node ---

type HTTPOption func(*models.Node)

func URL(url string) HTTPOption     { return func(n *models.Node) { n.Config["url"] = url } }
func Method(m string) HTTPOption    { return func(n *models.Node) { n.Config["method"] = m } }
func HTTPBody(body any) HTTPOption  { return func(n *models.Node) { n.Config["body"] = body } }
func HTTPTimeout(ms int) HTTPOption { return func(n *models.Node) { n.Config["timeout"] = ms } }

func Header(key, value string) HTTPOption {
	return func(n *models.Node) {
		headers, ok := n.Config["headers"].(map[string]string)
		if !ok {
			headers = make(map[string]string)
		}
		headers[key] = value
		n.Config["headers"] = headers
	}
}

func (b *WorkflowBuilder) AddHTTPNode(id, name string, opts ...HTTPOption) *WorkflowBuilder {
	nodeOpts := []NodeOption{func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		for _, o := range opts {
			o(n)
		}
	}}
	return b.AddNode(id, name, "http", nodeOpts...)
}

// --- LLM Node ---

type LLMOption func(*models.Node)

func Provider(p string) LLMOption      { return func(n *models.Node) { n.Config["provider"] = p } }
func Model(m string) LLMOption         { return func(n *models.Node) { n.Config["model"] = m } }
func Prompt(p string) LLMOption        { return func(n *models.Node) { n.Config["prompt"] = p } }
func APIKey(k string) LLMOption        { return func(n *models.Node) { n.Config["api_key"] = k } }
func Temperature(t float64) LLMOption  { return func(n *models.Node) { n.Config["temperature"] = t } }
func MaxTokens(m int) LLMOption        { return func(n *models.Node) { n.Config["max_tokens"] = m } }
func SystemPrompt(s string) LLMOption    { return func(n *models.Node) { n.Config["system_prompt"] = s } }
func ResponseSchema(s string) LLMOption  { return func(n *models.Node) { n.Config["response_schema"] = s } }

func (b *WorkflowBuilder) AddLLMNode(id, name string, opts ...LLMOption) *WorkflowBuilder {
	nodeOpts := []NodeOption{func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		for _, o := range opts {
			o(n)
		}
	}}
	return b.AddNode(id, name, "llm", nodeOpts...)
}

// --- Transform Node ---

type TransformOption func(*models.Node)

func TransformType(t string) TransformOption {
	return func(n *models.Node) { n.Config["type"] = t }
}

func TransformExpression(e string) TransformOption {
	return func(n *models.Node) { n.Config["expression"] = e }
}

func (b *WorkflowBuilder) AddTransformNode(id, name string, opts ...TransformOption) *WorkflowBuilder {
	nodeOpts := []NodeOption{func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		for _, o := range opts {
			o(n)
		}
	}}
	return b.AddNode(id, name, "transform", nodeOpts...)
}

// --- Conditional Node ---

type ConditionalOption func(*models.Node)

func Expression(e string) ConditionalOption {
	return func(n *models.Node) { n.Config["expression"] = e }
}

func (b *WorkflowBuilder) AddConditionalNode(id, name string, opts ...ConditionalOption) *WorkflowBuilder {
	nodeOpts := []NodeOption{func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		for _, o := range opts {
			o(n)
		}
	}}
	return b.AddNode(id, name, "conditional", nodeOpts...)
}

// --- Merge Node ---

type MergeOption func(*models.Node)

func MergeStrategy(s string) MergeOption {
	return func(n *models.Node) { n.Config["strategy"] = s }
}

func (b *WorkflowBuilder) AddMergeNode(id, name string, opts ...MergeOption) *WorkflowBuilder {
	nodeOpts := []NodeOption{func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		for _, o := range opts {
			o(n)
		}
	}}
	return b.AddNode(id, name, "merge", nodeOpts...)
}

// --- SubWorkflow Node ---

type SubWorkflowOption func(*models.Node)

func WorkflowID(id string) SubWorkflowOption {
	return func(n *models.Node) { n.Config["workflow_id"] = id }
}

func ForEach(expr string) SubWorkflowOption {
	return func(n *models.Node) { n.Config["for_each"] = expr }
}

func ItemVar(name string) SubWorkflowOption {
	return func(n *models.Node) { n.Config["item_var"] = name }
}

func MaxParallelism(n int) SubWorkflowOption {
	return func(nd *models.Node) { nd.Config["max_parallelism"] = n }
}

func OnError(strategy string) SubWorkflowOption {
	return func(n *models.Node) { n.Config["on_error"] = strategy }
}

func (b *WorkflowBuilder) AddSubWorkflowNode(id, name string, opts ...SubWorkflowOption) *WorkflowBuilder {
	nodeOpts := []NodeOption{func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		for _, o := range opts {
			o(n)
		}
	}}
	return b.AddNode(id, name, "sub_workflow", nodeOpts...)
}

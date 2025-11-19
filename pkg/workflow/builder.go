package workflow

type DefinitionBuilder struct {
	d Definition
}

func NewDefinitionBuilder() *DefinitionBuilder { return &DefinitionBuilder{d: Definition{}} }

func (b *DefinitionBuilder) Name(name string) *DefinitionBuilder { b.d.Name = name; return b }
func (b *DefinitionBuilder) Version(v string) *DefinitionBuilder { b.d.Version = v; return b }
func (b *DefinitionBuilder) Description(desc string) *DefinitionBuilder {
	b.d.Description = desc
	return b
}

func (b *DefinitionBuilder) AddTrigger(t TriggerDef) *DefinitionBuilder {
	b.d.Triggers = append(b.d.Triggers, t)
	return b
}

func (b *DefinitionBuilder) AddNode(n NodeDef) *DefinitionBuilder {
	b.d.Nodes = append(b.d.Nodes, n)
	return b
}

func (b *DefinitionBuilder) AddEdge(e EdgeDef) *DefinitionBuilder {
	b.d.Edges = append(b.d.Edges, e)
	return b
}

func (b *DefinitionBuilder) Build() Definition { return b.d }

// Convenience builders for elements
type NodeDefBuilder struct{ n NodeDef }

func NewNodeDefBuilder() *NodeDefBuilder                   { return &NodeDefBuilder{} }
func (b *NodeDefBuilder) ID(id string) *NodeDefBuilder     { b.n.ID = id; return b }
func (b *NodeDefBuilder) Type(t string) *NodeDefBuilder    { b.n.Type = t; return b }
func (b *NodeDefBuilder) Handler(h string) *NodeDefBuilder { b.n.Handler = h; return b }
func (b *NodeDefBuilder) Timeout(t string) *NodeDefBuilder { b.n.Timeout = t; return b }
func (b *NodeDefBuilder) Retry(max int, backoff string) *NodeDefBuilder {
	b.n.Retry = &RetryPolicy{MaxAttempts: max, Backoff: backoff}
	return b
}
func (b *NodeDefBuilder) ConfigKV(k string, v any) *NodeDefBuilder {
	if b.n.Config == nil {
		b.n.Config = map[string]any{}
	}
	b.n.Config[k] = v
	return b
}
func (b *NodeDefBuilder) Condition(expr string) *NodeDefBuilder { b.n.Condition = expr; return b }
func (b *NodeDefBuilder) Build() NodeDef                        { return b.n }

type EdgeDefBuilder struct{ e EdgeDef }

func NewEdgeDefBuilder() *EdgeDefBuilder                        { return &EdgeDefBuilder{} }
func (b *EdgeDefBuilder) From(id string) *EdgeDefBuilder        { b.e.From = id; return b }
func (b *EdgeDefBuilder) To(id string) *EdgeDefBuilder          { b.e.To = id; return b }
func (b *EdgeDefBuilder) Type(t string) *EdgeDefBuilder         { b.e.Type = t; return b }
func (b *EdgeDefBuilder) Condition(expr string) *EdgeDefBuilder { b.e.Condition = expr; return b }
func (b *EdgeDefBuilder) Transform(name string) *EdgeDefBuilder { b.e.Transform = name; return b }
func (b *EdgeDefBuilder) Build() EdgeDef                        { return b.e }

type TriggerDefBuilder struct{ t TriggerDef }

func NewTriggerDefBuilder() *TriggerDefBuilder                { return &TriggerDefBuilder{} }
func (b *TriggerDefBuilder) Type(t string) *TriggerDefBuilder { b.t.Type = t; return b }
func (b *TriggerDefBuilder) ID(id string) *TriggerDefBuilder  { b.t.ID = id; return b }
func (b *TriggerDefBuilder) ConfigKV(k string, v any) *TriggerDefBuilder {
	if b.t.Config == nil {
		b.t.Config = map[string]any{}
	}
	b.t.Config[k] = v
	return b
}
func (b *TriggerDefBuilder) Build() TriggerDef { return b.t }

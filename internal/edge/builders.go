package edge

import (
	n "mbflow/internal/node"
)

type DirectBuilder struct {
	from string
	to   string
}

func NewDirectBuilder() *DirectBuilder                 { return &DirectBuilder{} }
func (b *DirectBuilder) From(id string) *DirectBuilder { b.from = id; return b }
func (b *DirectBuilder) To(id string) *DirectBuilder   { b.to = id; return b }
func (b *DirectBuilder) Build() *Direct                { return NewDirect(b.from, b.to) }

type ConditionalBuilder struct {
	from string
	to   string
	cond ConditionFunc
}

func NewConditionalBuilder() *ConditionalBuilder                          { return &ConditionalBuilder{} }
func (b *ConditionalBuilder) From(id string) *ConditionalBuilder          { b.from = id; return b }
func (b *ConditionalBuilder) To(id string) *ConditionalBuilder            { b.to = id; return b }
func (b *ConditionalBuilder) When(cond ConditionFunc) *ConditionalBuilder { b.cond = cond; return b }
func (b *ConditionalBuilder) Build() *Conditional                         { return NewConditional(b.from, b.to, b.cond) }

// Helpers for simple conditions
func ConditionTrue() ConditionFunc { return func(_ n.NodeOutput) (bool, error) { return true, nil } }

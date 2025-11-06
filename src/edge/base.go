package edge

import (
	"mbflow/condition"
	"mbflow/node"
)

type IEdge[T any, U any] interface {
	GetSource() node.INode[T, U]
	GetTarget() node.INode[U, any]
	CanTransition(data U) bool
}

type BaseEdge[T any, U any] struct {
	Source    node.INode[T, U]
	Target    node.INode[U, any]
	Condition condition.ICondition[U]
}

func (e *BaseEdge[T, U]) GetSource() node.INode[T, U]   { return e.Source }
func (e *BaseEdge[T, U]) GetTarget() node.INode[U, any] { return e.Target }

func (e *BaseEdge[T, U]) CanTransition(data U) bool {
	if e.Condition == nil {
		return true
	}
	return e.Condition.Evaluate(data)
}

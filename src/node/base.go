package node

import (
	"fmt"
)

type Type string

const (
	TriggerNodeType Type = "Trigger"
)

// INode - базовый интерфейс ноды

type INode[T any, U any] interface {
	GetID() string
	GetType() Type
	GetName() string
	Execute(inputData T) (U, error)
}

// BaseNode - базовая структура для всех нод

type BaseNode[T any, U any] struct {
	ID   string
	Type Type
	Name string
}

func (n *BaseNode[T, U]) GetID() string   { return n.ID }
func (n *BaseNode[T, U]) GetType() Type   { return n.Type }
func (n *BaseNode[T, U]) GetName() string { return n.Name }

func (n *BaseNode[T, U]) Execute(inputData T) (U, error) {
	var output U
	return output, fmt.Errorf("Execute() not implemented for node type: %s", n.Type)
}

// Структура данных для TriggerNode

type TriggerEventData struct {
	EventName string
	Payload   map[string]any
}

// TriggerNode - стартовая нода (инициирует процесс)

type TriggerNode struct {
	BaseNode[TriggerEventData, TriggerEventData]
	EventType string
}

func NewTriggerNode(id, name, eventType string) *TriggerNode {
	return &TriggerNode{
		BaseNode: BaseNode[TriggerEventData, TriggerEventData]{
			ID:   id,
			Type: TriggerNodeType,
			Name: name,
		},
		EventType: eventType,
	}
}

func (t *TriggerNode) Execute(inputData TriggerEventData) (TriggerEventData, error) {
	return TriggerEventData{
		EventName: t.EventType,
		Payload:   inputData.Payload,
	}, nil
}

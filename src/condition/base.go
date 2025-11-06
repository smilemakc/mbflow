package condition

import "golang.org/x/exp/constraints"

type Number interface {
	constraints.Integer | constraints.Float
}

type ICondition[T any] interface {
	Evaluate(inputData T) bool
}

type BaseCondition[T any] struct{}

func (c *BaseCondition[T]) Evaluate(inputData T) bool {
	return false // Дефолтная реализация, всегда false
}

type EqualsCondition[T comparable] struct {
	Key   string
	Value T
}

func (c *EqualsCondition[T]) Evaluate(inputData map[string]any) bool {
	val, exists := inputData[c.Key]
	if !exists {
		return false
	}
	castedVal, ok := val.(T)
	return ok && castedVal == c.Value
}

type GreaterThanCondition[T ~string | Number] struct {
	Key   string
	Value T
}

func (c *GreaterThanCondition[T]) Evaluate(inputData map[string]any) bool {
	val, exists := inputData[c.Key]
	if !exists {
		return false
	}
	castedVal, ok := val.(T)
	return ok && castedVal > c.Value
}

type LessThanCondition[T ~string | Number] struct {
	Key   string
	Value T
}

func (c *LessThanCondition[T]) Evaluate(inputData map[string]any) bool {
	val, exists := inputData[c.Key]
	if !exists {
		return false
	}
	castedVal, ok := val.(T)
	return ok && castedVal < c.Value
}

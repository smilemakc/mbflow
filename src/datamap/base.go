package datamap

type IDataMapping[T any, U any] interface {
	Transform(inputData T) U
}

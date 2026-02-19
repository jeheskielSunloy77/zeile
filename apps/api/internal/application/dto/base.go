package dto

type BaseDTO[T any] interface {
	ToModel() *T
}

type StoreDTO[T any] interface {
	BaseDTO[T]
}

type UpdateDTO[T any] interface {
	BaseDTO[T]
	ToMap() map[string]any
}

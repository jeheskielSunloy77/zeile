package dto

import (
	"reflect"

	"github.com/jeheskielSunloy77/zeile/internal/interface/http/validation"
)

type Empty struct{}

func (d *Empty) Validate() error { return nil }

func NewDTO[T any]() T {
	var dto T
	t := reflect.TypeOf(dto)
	if t != nil && t.Kind() == reflect.Pointer {
		return reflect.New(t.Elem()).Interface().(T)
	}
	return dto
}

type StoreDTO[U any] interface {
	validation.Validatable
	ToUsecase() U
}

type UpdateDTO[U any] interface {
	validation.Validatable
	ToUsecase() U
}

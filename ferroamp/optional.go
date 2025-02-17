package ferroamp

import "errors"

type Optional[T any] struct {
	value T
	some  bool
}

func Some[T any](v T) Optional[T] {
	return Optional[T]{value: v, some: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{some: false}
}

func (o *Optional[T]) Get() (T, error) {
	if !o.some {
		return o.value, errors.New("the value is none")
	}
	return o.value, nil
}

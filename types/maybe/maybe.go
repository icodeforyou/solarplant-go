package maybe

type Maybe[T any] struct {
	value T
	valid bool
}

func Some[T any](value T) Maybe[T] {
	return Maybe[T]{
		value: value,
		valid: true,
	}
}

func None[T any]() Maybe[T] {
	return Maybe[T]{
		valid: false,
	}
}

func SqlNull[T any](value T, valid bool) Maybe[T] {
	return Maybe[T]{
		value: value,
		valid: valid,
	}
}

func (m Maybe[T]) IsValid() bool {
	return m.valid
}

func (m Maybe[T]) Value() T {
	return m.value
}

func (m Maybe[T]) ValueOrDefault(defaultValue T) T {
	if m.valid {
		return m.value
	}
	return defaultValue
}

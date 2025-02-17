// Maybe use package slices instead

package slice

func Map[T any, U any](input []T, pred func(T) U) []U {
	result := make([]U, len(input))
	for i, v := range input {
		result[i] = pred(v)
	}
	return result
}

func All[T any](input []T, pred func(T) bool) bool {
	for _, v := range input {
		if !pred(v) {
			return false
		}
	}
	return true
}

func Find[T any](input []T, pred func(T) bool) (T, bool) {
	for _, v := range input {
		if pred(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

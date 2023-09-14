package iterable

// Iterator returns items in a collection with every call to Next().
// The error will be set to io.EOF when the iterator is complete.
type Iterator[T any] interface {
	Next() (T, error)
}

type iterator[T any] struct {
	next func() (T, error)
}

func (it *iterator[T]) Next() (T, error) {
	return it.next()
}

func NewIterator[T any](next func() (T, error)) Iterator[T] {
	return &iterator[T]{next}
}

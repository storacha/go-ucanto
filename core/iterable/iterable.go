package iterable

// Iterator returns items in a collection with every call to Next().
// The error will be set to io.EOF when the iterator is complete.
type Iterator[T any] interface {
	Next() (T, error)
}

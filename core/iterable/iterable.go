package iterable

import (
	"io"
)

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

func From[Slice ~[]T, T any](slice Slice) Iterator[T] {
	i := 0
	return NewIterator(func() (T, error) {
		if i < len(slice) {
			item := slice[i]
			i++
			return item, nil
		}
		var undef T
		return undef, io.EOF
	})
}

func Collect[T any](it Iterator[T]) ([]T, error) {
	var items []T
	for {
		item, err := it.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func Map[T, U any](it Iterator[T], mapFn func(T) U) Iterator[U] {
	return NewIterator(func() (U, error) {
		t, err := it.Next()
		if err != nil {
			var undef U
			return undef, err
		}
		return mapFn(t), nil
	})
}

func Concat[T any](iterators ...Iterator[T]) Iterator[T] {
	if len(iterators) == 0 {
		return From([]T{})
	}

	i := 0
	iterator := iterators[i]
	return NewIterator(func() (T, error) {
		for {
			item, err := iterator.Next()
			if err != nil {
				if err == io.EOF {
					i++
					if i < len(iterators) {
						iterator = iterators[i]
						continue
					}
				}
				var undef T
				return undef, err
			}
			return item, nil
		}
	})
}

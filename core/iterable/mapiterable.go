package iterable

import "io"

// Iterator2 returns two values with every call to Next().
// The error will be set to io.EOF when the iterator is complete.
type Iterator2[K any, V any] interface {
	Next() (K, V, error)
}

type iterator2[K any, V any] struct {
	next func() (K, V, error)
}

func (mit *iterator2[K, V]) Next() (K, V, error) {
	return mit.next()
}

func NewIterator2[K any, V any](next func() (K, V, error)) Iterator2[K, V] {
	return &iterator2[K, V]{next}
}

type mapEntry[K comparable, V any] struct {
	k K
	v V
}

func FromMap[Map ~map[K]V, K comparable, V any](m Map) Iterator2[K, V] {
	entries := make([]mapEntry[K, V], 0, len(m))
	for k, v := range m {
		entries = append(entries, mapEntry[K, V]{k, v})
	}
	i := 0
	return NewIterator2(func() (K, V, error) {
		if i < len(entries) {
			k := entries[i].k
			v := entries[i].v
			i++
			return k, v, nil
		}
		var k K
		var v V
		return k, v, io.EOF
	})
}

func CollectMap[K comparable, V any](mit Iterator2[K, V]) (map[K]V, error) {
	items := make(map[K]V)
	for {
		k, v, err := mit.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		items[k] = v
	}
	return items, nil
}

func Concat2[K any, V any](iterators ...Iterator2[K, V]) Iterator2[K, V] {
	if len(iterators) == 0 {
		return NewIterator2(func() (K, V, error) {
			var k K
			var v V
			return k, v, io.EOF
		})
	}

	i := 0
	iterator := iterators[i]
	return NewIterator2(func() (K, V, error) {
		for {
			k, v, err := iterator.Next()
			if err != nil {
				if err == io.EOF {
					i++
					if i < len(iterators) {
						iterator = iterators[i]
						continue
					}
				}
				var k K
				var v V
				return k, v, err
			}
			return k, v, nil
		}
	})
}

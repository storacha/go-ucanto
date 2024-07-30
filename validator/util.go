package validator

func combine[T any](dataset [][]T) [][]T {
	first, rest := dataset[0], dataset[1:]
	results := make([][]T, 0, len(first))
	for _, value := range first {
		results = append(results, []T{value})
	}
	for _, values := range rest {
		tuples := results
		results = make([][]T, 0, len(tuples))
		for _, value := range values {
			for _, tuple := range tuples {
				newTuple := make([]T, len(tuple), len(tuple)+1)
				_ = copy(newTuple, tuple)
				results = append(results, append(newTuple, value))
			}
		}
	}
	return results
}

func intersection[T comparable](left []T, right []T) []T {
	set := make([]T, 0)
	hash := make(map[T]struct{})

	for _, v := range left {
		hash[v] = struct{}{}
	}

	for _, v := range right {
		if _, ok := hash[v]; ok {
			set = append(set, v)
		}
	}

	return set
}

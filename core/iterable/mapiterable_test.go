package iterable_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/storacha-network/go-ucanto/core/iterable"
	"github.com/stretchr/testify/require"
)

func TestCollectMap(t *testing.T) {
	someErr := errors.New("some error")
	testCases := []struct {
		name        string
		iterator2   func() iterable.Iterator2[string, int]
		expectedMap map[string]int
		expectedErr error
	}{
		{
			name: "converts successful iterator to expected map",
			iterator2: func() iterable.Iterator2[string, int] {
				count := 0
				return iterable.NewIterator2(func() (string, int, error) {
					defer func() {
						count++
					}()
					switch count {
					case 0:
						return "apples", 7, nil
					case 1:
						return "oranges", 4, nil
					case 2:
						return "", 0, io.EOF
					default:
						return "", 0, fmt.Errorf("too many calls to iterator: %d", count+1)
					}
				})
			},
			expectedMap: map[string]int{"apples": 7, "oranges": 4},
		},
		{
			name: "fails when iterator fails",
			iterator2: func() iterable.Iterator2[string, int] {
				count := 0
				return iterable.NewIterator2(func() (string, int, error) {
					defer func() {
						count++
					}()
					switch count {
					case 0:
						return "apples", 7, nil
					default:
						return "", 0, fmt.Errorf("mistake iterating: %w", someErr)
					}
				})
			},
			expectedMap: nil,
			expectedErr: someErr,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resultMap, err := iterable.CollectMap(testCase.iterator2())
			require.Equal(t, testCase.expectedMap, resultMap)
			require.ErrorIs(t, err, testCase.expectedErr)
		})
	}
}

func TestFromMap(t *testing.T) {
	verifyFromMap(t, "string -> int", map[string]int{"apples": 7, "oranges": 4})
	verifyFromMap(t, "int -> bool", map[int]bool{7: true, 4: false, 3: true})
	verifyFromMap(t, "any -> any", map[any]any{
		7:         true,
		4:         "apples",
		"oranges": struct{ head string }{head: "bucket"},
	})
}

func TestRoundtrip(t *testing.T) {
	roundTrip(t, "string -> int", map[string]int{"apples": 7, "oranges": 4})
	roundTrip(t, "int -> bool", map[int]bool{7: true, 4: false, 3: true})
	roundTrip(t, "any -> any", map[any]any{
		7:         true,
		4:         "apples",
		"oranges": struct{ head string }{head: "bucket"},
	})
}

func verifyFromMap[K comparable, V any](t *testing.T, testCase string, inputMap map[K]V) {
	t.Run(testCase, func(t *testing.T) {
		iterator := iterable.FromMap(inputMap)
		outputMap := make(map[K]V, len(inputMap))
		for {
			k, v, err := iterator.Next()
			if err != nil {
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, inputMap, outputMap)
				return
			}
			outputMap[k] = v
		}
	})
}

func roundTrip[K comparable, V any](t *testing.T, testCase string, inputMap map[K]V) {
	t.Run(testCase, func(t *testing.T) {
		outputMap, err := iterable.CollectMap(iterable.FromMap(inputMap))
		require.NoError(t, err)
		require.Equal(t, inputMap, outputMap)
	})
}

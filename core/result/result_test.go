package result_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/stretchr/testify/require"
)

func TestMatchResult(t *testing.T) {
	t.Run("MatchResultR0", func(t *testing.T) {
		testMatchResultR0(t, "ok (int)", result.Ok[int, any](5), true, false)
		testMatchResultR0(t, "ok (string)", result.Ok[string, any]("apple"), true, false)
		testMatchResultR0(t, "err (error)", result.Error[int](errors.New("bad")), false, true)
	})
	t.Run("MatchResultR1", func(t *testing.T) {
		testMatchResultR1(t, "ok (int)",
			result.Ok[int, any](5),
			func(o int) int { return o * 2 },
			func(x any) int { return 0 },
			10)
		testMatchResultR1(t, "ok (string)",
			result.Ok[string, any]("apple"),
			func(o string) string { return o + " tree" },
			func(x any) string { return "nothing" },
			"apple tree")
		testMatchResultR1(t, "err (error)",
			result.Error[int](errors.New("bad")),
			func(o int) string { return "" },
			func(x error) string { return x.Error() },
			"bad")
	})
	t.Run("MatchResultR2", func(t *testing.T) {
		testMatchResultR2(t, "ok (int)",
			result.Ok[int, any](5),
			func(o int) (int, int) { return o * 2, o * 3 },
			func(x any) (int, int) { return 0, 0 },
			10, 15)
		testMatchResultR2(t, "ok (string)",
			result.Ok[string, any]("apple"),
			func(o string) (string, int) { return o + " tree", len(o) },
			func(x any) (string, int) { return "nothing", 0 },
			"apple tree", 5)
		testMatchResultR2(t, "err (error)",
			result.Error[int](errors.New("bad")),
			func(o int) (string, error) { return "", nil },
			func(x error) (string, error) { return x.Error(), fmt.Errorf("something: %w", x) },
			"bad", fmt.Errorf("something: %w", errors.New("bad")))
	})

	t.Run("MatchResultR2", func(t *testing.T) {
		testMatchResultR3(t, "ok (int)",
			result.Ok[int, any](5),
			func(o int) (int, int, int) { return o * 2, o * 3, o * 4 },
			func(x any) (int, int, int) { return 0, 0, 0 },
			10, 15, 20)
		testMatchResultR3(t, "ok (string)",
			result.Ok[string, any]("apple"),
			func(o string) (string, int, string) { return o + " tree", len(o), o + " cart" },
			func(x any) (string, int, string) { return "nothing", 0, "nothing" },
			"apple tree", 5, "apple cart")
		testMatchResultR3(t, "err (error)",
			result.Error[int](errors.New("bad")),
			func(o int) (string, error, int) { return "", nil, 0 },
			func(x error) (string, error, int) { return x.Error(), fmt.Errorf("something: %w", x), len(x.Error()) },
			"bad", fmt.Errorf("something: %w", errors.New("bad")), 3)
	})
}

func TestMap(t *testing.T) {
	t.Run("MapOk", func(t *testing.T) {
		testMapOk(t, "ok (int)", result.Ok[int, error](5), func(o int) int { return o * 2 }, result.Ok[int, error](10))
		testMapOk(t, "ok (string)",
			result.Ok[string, error]("apple"),
			func(o string) int { return len(o) },
			result.Ok[int, error](5))
		testMapOk(t, "err (int)",
			result.Error[int](errors.New("bad")),
			func(o int) int { return o * 2 },
			result.Error[int](errors.New("bad")))
		testMapOk(t, "err (string)",
			result.Error[string](errors.New("bad")),
			func(o string) int { return len(o) },
			result.Error[int](errors.New("bad")))
	})
	t.Run("MapError", func(t *testing.T) {
		testMapErr(t, "ok (int)",
			result.Ok[int, error](5),
			func(e error) error { return fmt.Errorf("something: %w", e) },
			result.Ok[int, error](5))
		testMapErr(t, "ok (string)",
			result.Ok[string, error]("apple"),
			func(e error) string { return e.Error() },
			result.Ok[string, string]("apple"))
		testMapErr(t, "err (int)",
			result.Error[int](errors.New("bad")),
			func(e error) error { return fmt.Errorf("something: %w", e) },
			result.Error[int](fmt.Errorf("something: %w", errors.New("bad"))))
		testMapErr(t, "err (string)",
			result.Error[string](errors.New("bad")),
			func(e error) string { return e.Error() },
			result.Error[string]("bad"))
	})
	t.Run("MapResult", func(t *testing.T) {
		testMapResult(t, "ok (int)",
			result.Ok[int, error](5),
			func(o int) int { return o * 2 },
			func(e error) error { return fmt.Errorf("something: %w", e) },
			result.Ok[int, error](10))
		testMapResult(t, "ok (string)",
			result.Ok[string, error]("apple"),
			func(o string) int { return len(o) },
			func(e error) string { return e.Error() },
			result.Ok[int, string](5))
		testMapResult(t, "err (int)",
			result.Error[int](errors.New("bad")),
			func(o int) int { return o * 2 },
			func(e error) error { return fmt.Errorf("something: %w", e) },
			result.Error[int](fmt.Errorf("something: %w", errors.New("bad"))))
		testMapResult(t, "err (string)",
			result.Error[string](errors.New("bad")),
			func(o string) int { return len(o) },
			func(e error) string { return e.Error() },
			result.Error[int]("bad"))
	})

	t.Run("MapResult", func(t *testing.T) {
		testMapResultR1(t, "ok (int)",
			result.Ok[int, error](5),
			func(o int) (int, error) { return o * 2, nil },
			func(e error) (error, error) { return fmt.Errorf("something: %w", e), errors.New("very") },
			result.Ok[int, error](10), nil)
		testMapResultR1(t, "ok (string)",
			result.Ok[string, error]("apple"),
			func(o string) (int, string) { return len(o), o + " tree" },
			func(e error) (string, string) { return e.Error(), fmt.Errorf("something: %w", e).Error() },
			result.Ok[int, string](5), "apple tree")
		testMapResultR1(t, "err (int)",
			result.Error[int](errors.New("bad")),
			func(o int) (int, error) { return o * 2, nil },
			func(e error) (error, error) { return fmt.Errorf("something: %w", e), errors.New("very") },
			result.Error[int](fmt.Errorf("something: %w", errors.New("bad"))), errors.New("very"))
		testMapResultR1(t, "err (string)",
			result.Error[string](errors.New("bad")),
			func(o string) (int, string) { return len(o), o + " tree" },
			func(e error) (string, string) { return e.Error(), fmt.Errorf("something: %w", e).Error() },
			result.Error[int]("bad"), "something: bad")
	})
}

func TestWrap(t *testing.T) {
	require.Equal(t,
		result.Ok[int, error](5),
		result.Wrap(func() (int, error) { return 5, nil }),
		"int (no error)")
	require.Equal(t,
		result.Error[int](errors.New("bad")),
		result.Wrap(func() (int, error) { return 0, errors.New("bad") }),
		"string (no error)")
	require.Equal(t,
		result.Ok[string, error]("apple"),
		result.Wrap(func() (string, error) { return "apple", nil }),
		"int (no error)")
	require.Equal(t,
		result.Error[string](errors.New("bad")),
		result.Wrap(func() (string, error) { return "", errors.New("bad") }),
		"string (error present)")
}

func TestAndOr(t *testing.T) {
	t.Run("And", func(t *testing.T) {
		testAnd(t, "O - int, O2 - string, X - error (no error)",
			result.Ok[int, error](10),
			result.Ok[string, error]("apple"),
			result.Ok[string, error]("apple"))
		testAnd(t, "O - int, O2 - string, X - error (first error)",
			result.Error[int](errors.New("bad")),
			result.Ok[string, error]("apple"),
			result.Error[string](errors.New("bad")))
		testAnd(t, "O - int, O2 - string, X - error (second error)",
			result.Ok[int, error](10),
			result.Error[string](errors.New("very bad")),
			result.Error[string](errors.New("very bad")))
		testAnd(t, "O - int, O2 - string, X - error (both error)",
			result.Error[int](errors.New("bad")),
			result.Error[string](errors.New("very bad")),
			result.Error[string](errors.New("bad")))
	})

	t.Run("Or", func(t *testing.T) {
		testOr(t, "O - int, X - error, X2 - string (no error)",
			result.Ok[int, error](10),
			result.Ok[int, string](5),
			result.Ok[int, string](10))
		testOr(t, "O - int, X - error, X2 - string (first error)",
			result.Error[int](errors.New("bad")),
			result.Ok[int, string](5),
			result.Ok[int, string](5))
		testOr(t, "O - int, X - error, X2 - string (second error)",
			result.Ok[int, error](10),
			result.Error[int]("very bad"),
			result.Ok[int, string](10))
		testOr(t, "O - int, X - error, X2 - string (both error)",
			result.Error[int](errors.New("bad")),
			result.Error[int]("very bad"),
			result.Error[int]("very bad"))
	})

	t.Run("AndThen", func(t *testing.T) {
		testAndThen(t, "O - int, O2 - string, X - error (ok, () -> ok)",
			result.Ok[int, error](10),
			func(x int) result.Result[string, error] {
				return result.Ok[string, error](fmt.Sprintf("%d", x))
			},
			result.Ok[string, error]("10"))
		testAndThen(t, "O - int, O2 - string, X - error (err, () -> ok)",
			result.Error[int](errors.New("bad")),
			func(x int) result.Result[string, error] {
				return result.Ok[string, error](fmt.Sprintf("%d", x))
			},
			result.Error[string](errors.New("bad")))
		testAndThen(t, "O - int, O2 - string, X - error (ok, () -> err)",
			result.Ok[int, error](10),
			func(x int) result.Result[string, error] {
				return result.Error[string](fmt.Errorf("%d", x))
			},
			result.Error[string](fmt.Errorf("10")))
		testAndThen(t, "O - int, O2 - string, X - error (err, () -> err)",
			result.Error[int](errors.New("bad")),
			func(x int) result.Result[string, error] {
				return result.Error[string](fmt.Errorf("%d", x))
			},
			result.Error[string](errors.New("bad")))
	})
	t.Run("OrElse", func(t *testing.T) {
		testOrElse(t, "O - int, X - error, X2 - string (ok, () -> ok)",
			result.Ok[int, error](10),
			func(e error) result.Result[int, string] {
				return result.Ok[int, string](len(e.Error()))
			},
			result.Ok[int, string](10))
		testOrElse(t, "O - int, X - error, X2 - string (err, () -> ok)",
			result.Error[int](errors.New("bad")),
			func(e error) result.Result[int, string] {
				return result.Ok[int, string](len(e.Error()))
			},
			result.Ok[int, string](3))
		testOrElse(t, "O - int, X - error, X2 - string (ok, () -> err)",
			result.Ok[int, error](10),
			func(e error) result.Result[int, string] {
				return result.Error[int](e.Error())
			},
			result.Ok[int, string](10))
		testOrElse(t, "O - int, X - error, X2 - string (err, () -> err)",
			result.Error[int](errors.New("bad")),
			func(e error) result.Result[int, string] {
				return result.Error[int](e.Error())
			},
			result.Error[int]("bad"))
	})
}
func testMatchResultR0[X, O any](t *testing.T, testCase string, testResult result.Result[O, X], expOk bool, expErr bool) {
	t.Run(testCase, func(t *testing.T) {
		var okCalled, errCalled bool
		result.MatchResultR0(testResult, func(_ O) {
			okCalled = true
		}, func(_ X) {
			errCalled = true
		})
		require.Equal(t, expOk, okCalled)
		require.Equal(t, expErr, errCalled)
	})
}

func testMatchResultR1[X, O, R1 any](t *testing.T, testCase string, testResult result.Result[O, X], onOk func(O) R1, onErr func(X) R1, expR1 R1) {
	t.Run(testCase, func(t *testing.T) {
		r1 := result.MatchResultR1(testResult, onOk, onErr)
		require.Equal(t, expR1, r1)
	})
}

func testMatchResultR2[X, O, R1, R2 any](t *testing.T, testCase string, testResult result.Result[O, X], onOk func(O) (R1, R2), onErr func(X) (R1, R2), expR1 R1, expR2 R2) {
	t.Run(testCase, func(t *testing.T) {
		r1, r2 := result.MatchResultR2(testResult, onOk, onErr)
		require.Equal(t, expR1, r1)
		require.Equal(t, expR2, r2)
	})
}

func testMatchResultR3[X, O, R1, R2, R3 any](
	t *testing.T,
	testCase string,
	testResult result.Result[O, X],
	onOk func(O) (R1, R2, R3),
	onErr func(X) (R1, R2, R3),
	expR1 R1,
	expR2 R2,
	expR3 R3,
) {
	t.Run(testCase, func(t *testing.T) {
		r1, r2, r3 := result.MatchResultR3(testResult, onOk, onErr)
		require.Equal(t, expR1, r1)
		require.Equal(t, expR2, r2)
		require.Equal(t, expR3, r3)
	})
}

func testMapOk[O, O2, X any](t *testing.T,
	testCase string,
	testResult result.Result[O, X],
	mapFn func(O) O2,
	expResult result.Result[O2, X]) {
	t.Run(testCase, func(t *testing.T) {
		result := result.MapOk(testResult, mapFn)
		require.Equal(t, expResult, result)
	})
}

func testMapErr[O, X, X2 any](t *testing.T,
	testCase string,
	testResult result.Result[O, X],
	mapFn func(X) X2,
	expResult result.Result[O, X2]) {
	t.Run(testCase, func(t *testing.T) {
		result := result.MapError(testResult, mapFn)
		require.Equal(t, expResult, result)
	})
}

func testMapResult[O, O2, X, X2 any](t *testing.T,
	testCase string,
	testResult result.Result[O, X],
	mapOk func(O) O2,
	mapErr func(X) X2,
	expResult result.Result[O2, X2]) {
	t.Run(testCase, func(t *testing.T) {
		result := result.MapResultR0(testResult, mapOk, mapErr)
		require.Equal(t, expResult, result)
	})
}

func testMapResultR1[O, O2, X, X2, R1 any](t *testing.T,
	testCase string,
	testResult result.Result[O, X],
	mapOk func(O) (O2, R1),
	mapErr func(X) (X2, R1),
	expResult result.Result[O2, X2],
	expR1 R1) {
	t.Run(testCase, func(t *testing.T) {
		result, r1 := result.MapResultR1(testResult, mapOk, mapErr)
		require.Equal(t, expResult, result)
		require.Equal(t, expR1, r1)
	})
}

func testAnd[O, O2, X any](
	t *testing.T,
	testCase string,
	r1 result.Result[O, X],
	r2 result.Result[O2, X],
	expResult result.Result[O2, X]) {
	t.Run(testCase, func(t *testing.T) {
		require.Equal(t, expResult, result.And(r1, r2))
	})
}

func testOr[O, X, X2 any](
	t *testing.T,
	testCase string,
	r1 result.Result[O, X],
	r2 result.Result[O, X2],
	expResult result.Result[O, X2]) {
	t.Run(testCase, func(t *testing.T) {
		require.Equal(t, expResult, result.Or(r1, r2))
	})
}

func testAndThen[O, O2, X any](
	t *testing.T,
	testCase string,
	r1 result.Result[O, X],
	after func(O) result.Result[O2, X],
	expResult result.Result[O2, X]) {
	t.Run(testCase, func(t *testing.T) {
		require.Equal(t, expResult, result.AndThen(r1, after))
	})
}

func testOrElse[O, X, X2 any](
	t *testing.T,
	testCase string,
	r1 result.Result[O, X],
	after func(X) result.Result[O, X2],
	expResult result.Result[O, X2]) {
	t.Run(testCase, func(t *testing.T) {
		require.Equal(t, expResult, result.OrElse(r1, after))
	})
}

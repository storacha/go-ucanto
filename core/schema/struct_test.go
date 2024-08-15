package schema

import (
	"fmt"
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func TestReadStruct(t *testing.T) {
	type TestStruct struct {
		Name string
	}

	ts := helpers.Must(ipld.LoadSchemaBytes([]byte(`
		type TestStruct struct {
			name String
		}
	`)))

	t.Run("Success", func(t *testing.T) {
		np := basicnode.Prototype.Any
		nb := np.NewBuilder()
		ma := helpers.Must(nb.BeginMap(2))
		ma.AssembleKey().AssignString("name")
		ma.AssembleValue().AssignString("foo")
		ma.Finish()
		nd := nb.Build()

		res := Struct[TestStruct](ts.TypeByName("TestStruct")).Read(nd)
		result.MatchResultR0(res, func(ok TestStruct) {
			fmt.Printf("%+v\n", ok)
			require.Equal(t, ok.Name, "foo")
		}, func(err result.Failure) {
			t.Fatalf("unexpected error reading struct: %s", err)
		})
	})

	t.Run("Failure", func(t *testing.T) {
		np := basicnode.Prototype.Any
		nb := np.NewBuilder()
		ma := helpers.Must(nb.BeginMap(2))
		ma.AssembleKey().AssignString("foo")
		ma.AssembleValue().AssignString("bar")
		ma.Finish()
		nd := nb.Build()

		res := Struct[TestStruct](ts.TypeByName("TestStruct")).Read(nd)
		result.MatchResultR0(res, func(ok TestStruct) {
			t.Fatalf("unexpectedly read incompatible struct: %+v", ok)
		}, func(err result.Failure) {
			fmt.Printf("%+v\n", err)
			require.Equal(t, err.Name(), "SchemaError")
		})
	})
}

package schema

import (
	"fmt"
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/node/basicnode"
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

		res, err := Struct[TestStruct](ts.TypeByName("TestStruct"), nil).Read(nd)
		require.NoError(t, err)
		fmt.Printf("%+v\n", res)
		require.Equal(t, res.Name, "foo")
	})

	t.Run("Failure", func(t *testing.T) {
		np := basicnode.Prototype.Any
		nb := np.NewBuilder()
		ma := helpers.Must(nb.BeginMap(2))
		ma.AssembleKey().AssignString("foo")
		ma.AssembleValue().AssignString("bar")
		ma.Finish()
		nd := nb.Build()

		_, err := Struct[TestStruct](ts.TypeByName("TestStruct"), nil).Read(nd)
		require.Error(t, err)
		fmt.Printf("%+v\n", err)
		require.Equal(t, err.Name(), "SchemaError")
	})
}

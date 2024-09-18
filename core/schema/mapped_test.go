package schema_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func TestReadStruct(t *testing.T) {
	type TestStruct struct {
		Url string
	}

	ts := helpers.Must(ipld.LoadSchemaBytes([]byte(`
		type TestStruct struct {
			url String
		}
	`)))

	type URLStruct struct {
		Url url.URL
	}

	converter := func(ts TestStruct) (URLStruct, failure.Failure) {
		url, err := url.Parse(ts.Url)
		if err != nil {
			return URLStruct{}, failure.FromError(err)
		}
		return URLStruct{Url: *url}, nil
	}

	t.Run("Success", func(t *testing.T) {
		np := basicnode.Prototype.Any
		nb := np.NewBuilder()
		ma := helpers.Must(nb.BeginMap(2))
		ma.AssembleKey().AssignString("url")
		ma.AssembleValue().AssignString("http://www.yahoo.com")
		ma.Finish()
		nd := nb.Build()

		res, err := schema.Mapped(schema.Struct[TestStruct](ts.TypeByName("TestStruct"), nil), converter).Read(nd)
		require.NoError(t, err)
		fmt.Printf("%+v\n", res)
		require.Equal(t, res.Url.Host, "www.yahoo.com")
	})

	t.Run("Failure, underlying reader", func(t *testing.T) {
		np := basicnode.Prototype.Any
		nb := np.NewBuilder()
		ma := helpers.Must(nb.BeginMap(2))
		ma.AssembleKey().AssignString("foo")
		ma.AssembleValue().AssignString("bar")
		ma.Finish()
		nd := nb.Build()

		_, err := schema.Mapped(schema.Struct[TestStruct](ts.TypeByName("TestStruct"), nil), converter).Read(nd)
		require.Error(t, err)
		fmt.Printf("%+v\n", err)
		require.Equal(t, err.Name(), "SchemaError")
	})
	t.Run("Failure, conversion", func(t *testing.T) {
		np := basicnode.Prototype.Any
		nb := np.NewBuilder()
		ma := helpers.Must(nb.BeginMap(2))
		ma.AssembleKey().AssignString("url")
		ma.AssembleValue().AssignString(":apple")
		ma.Finish()
		nd := nb.Build()

		_, err := schema.Mapped(schema.Struct[TestStruct](ts.TypeByName("TestStruct"), nil), converter).Read(nd)
		require.Error(t, err)
		fmt.Printf("%+v\n", err)
	})
}

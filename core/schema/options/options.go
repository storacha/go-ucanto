package options

import (
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

// NamedBoolConverter is a generic version of bindnode.NamedBoolConverter
func NamedBoolConverter[T any](typeName schema.TypeName, from func(bool) (T, error), to func(T) (bool, error)) bindnode.Option {
	return bindnode.NamedBoolConverter(typeName, func(b bool) (interface{}, error) { return from(b) }, func(t interface{}) (bool, error) { return to(t.(T)) })
}

// NamedIntConverter is a generic version of bindnode.NamedIntConverter
func NamedIntConverter[T any](typeName schema.TypeName, from func(int64) (T, error), to func(T) (int64, error)) bindnode.Option {
	return bindnode.NamedIntConverter(typeName, func(i int64) (interface{}, error) { return from(i) }, func(t interface{}) (int64, error) { return to(t.(T)) })
}

// NamedFloatConverter is a generic version of bindnode.NamedFloatConverter
func NamedFloatConverter[T any](typeName schema.TypeName, from func(float64) (T, error), to func(T) (float64, error)) bindnode.Option {
	return bindnode.NamedFloatConverter(typeName, func(f float64) (interface{}, error) { return from(f) }, func(t interface{}) (float64, error) { return to(t.(T)) })
}

// NamedStringConverter is a generic version of bindnode.NamedStringConverter
func NamedStringConverter[T any](typeName schema.TypeName, from func(string) (T, error), to func(T) (string, error)) bindnode.Option {
	return bindnode.NamedStringConverter(typeName, func(s string) (interface{}, error) { return from(s) }, func(t interface{}) (string, error) { return to(t.(T)) })
}

// NamedBytesConverter is a generic version of bindnode.NamedBytesConverter
func NamedBytesConverter[T any](typeName schema.TypeName, from func([]byte) (T, error), to func(T) ([]byte, error)) bindnode.Option {
	return bindnode.NamedBytesConverter(typeName, func(b []byte) (interface{}, error) { return from(b) }, func(t interface{}) ([]byte, error) { return to(t.(T)) })
}

// NamedLinkConverter is a generic version of bindnode.NamedLinkConverter
func NamedLinkConverter[T any](typeName schema.TypeName, from func(cid.Cid) (T, error), to func(T) (cid.Cid, error)) bindnode.Option {
	return bindnode.NamedLinkConverter(typeName, func(c cid.Cid) (interface{}, error) { return from(c) }, func(t interface{}) (cid.Cid, error) { return to(t.(T)) })
}

// NamedAnyConverter is a generic version of bindnode.NamedAnyConverter
func NamedAnyConverter[T any](typeName schema.TypeName, from func(datamodel.Node) (T, error), to func(T) (datamodel.Node, error)) bindnode.Option {
	return bindnode.NamedAnyConverter(typeName, func(n datamodel.Node) (interface{}, error) { return from(n) }, func(t interface{}) (datamodel.Node, error) { return to(t.(T)) })
}

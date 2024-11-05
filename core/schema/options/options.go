package options

import (
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

// NamedBoolConverter is a generic version of bindnode.NamedBoolConverter
func NamedBoolConverter[T any](typeName schema.TypeName, from func(bool) (T, error), to func(T) (bool, error)) bindnode.Option {
	return bindnode.NamedBoolConverter(typeName, func(b bool) (interface{}, error) {
		t, err := from(b)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) (bool, error) {
		as := t.(*T)
		if as == nil {
			return false, nil
		}
		return to(*as)
	})
}

// NamedIntConverter is a generic version of bindnode.NamedIntConverter
func NamedIntConverter[T any](typeName schema.TypeName, from func(int64) (T, error), to func(T) (int64, error)) bindnode.Option {
	return bindnode.NamedIntConverter(typeName, func(i int64) (interface{}, error) {
		t, err := from(i)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) (int64, error) {
		as := t.(*T)
		if as == nil {
			return 0, nil
		}
		return to(*as)
	})
}

// NamedFloatConverter is a generic version of bindnode.NamedFloatConverter
func NamedFloatConverter[T any](typeName schema.TypeName, from func(float64) (T, error), to func(T) (float64, error)) bindnode.Option {
	return bindnode.NamedFloatConverter(typeName, func(f float64) (interface{}, error) {
		t, err := from(f)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) (float64, error) {
		as := t.(*T)
		if as == nil {
			return 0, nil
		}
		return to(*as)
	})
}

// NamedStringConverter is a generic version of bindnode.NamedStringConverter
func NamedStringConverter[T any](typeName schema.TypeName, from func(string) (T, error), to func(T) (string, error)) bindnode.Option {
	return bindnode.NamedStringConverter(typeName, func(s string) (interface{}, error) {
		t, err := from(s)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) (string, error) {
		as := t.(*T)
		if as == nil {
			return "", nil
		}
		return to(*as)
	})
}

// NamedBytesConverter is a generic version of bindnode.NamedBytesConverter
func NamedBytesConverter[T any](typeName schema.TypeName, from func([]byte) (T, error), to func(T) ([]byte, error)) bindnode.Option {
	return bindnode.NamedBytesConverter(typeName, func(b []byte) (interface{}, error) {
		t, err := from(b)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) ([]byte, error) {
		as := t.(*T)
		if as == nil {
			return nil, nil
		}
		return to(*as)
	})
}

// NamedLinkConverter is a generic version of bindnode.NamedLinkConverter
func NamedLinkConverter[T any](typeName schema.TypeName, from func(cid.Cid) (T, error), to func(T) (cid.Cid, error)) bindnode.Option {
	return bindnode.NamedLinkConverter(typeName, func(c cid.Cid) (interface{}, error) {
		t, err := from(c)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) (cid.Cid, error) {
		as := t.(*T)
		if as == nil {
			return cid.Undef, nil
		}
		return to(*as)
	})
}

// NamedAnyConverter is a generic version of bindnode.NamedAnyConverter
func NamedAnyConverter[T any](typeName schema.TypeName, from func(datamodel.Node) (T, error), to func(T) (datamodel.Node, error)) bindnode.Option {
	return bindnode.NamedAnyConverter(typeName, func(n datamodel.Node) (interface{}, error) {
		t, err := from(n)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}, func(t interface{}) (datamodel.Node, error) {
		as := t.(*T)
		if as == nil {
			return datamodel.Null, nil
		}
		return to(*as)
	})
}

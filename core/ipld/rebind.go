package ipld

import (
	"reflect"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

// Rebind takes a Node and binds it to the Go type according to the passed schema.
func Rebind(nd datamodel.Node, ptrVal any, typ schema.Type) (datamodel.Node, error) {
	np := bindnode.Prototype(ptrVal, typ)
	nb := np.Representation().NewBuilder()
	nb.AssignNode(nd)
	nd = nb.Build()

	// Code and comments below are from UnmarshalStreaming...
	// https://github.com/ipld/go-ipld-prime/blob/36adac0f53c70d7fab5131c4295054463b7b6cb3/codecHelpers.go#L161-L168

	// ... but our approach above allocated new memory, and we have to copy it back out.
	// In the future, the bindnode API could be improved to make this easier.
	if !reflect.ValueOf(ptrVal).IsNil() {
		reflect.ValueOf(ptrVal).Elem().Set(reflect.ValueOf(bindnode.Unwrap(nd)).Elem())
	}
	// ... and we also have to re-bind a new node to the 'bind' value,
	// because probably the user will be surprised if mutating 'bind' doesn't affect the Node later.
	nd = bindnode.Wrap(ptrVal, typ)
	return nd, nil
}

// func doBind(nb datamodel.NodeAssembler, nd datamodel.Node) error {
// 	switch nd.Kind() {
// 	case datamodel.Kind_Map:
// 		fmt.Println("FOUND MAP")
// 		ma, err := nb.BeginMap(1)
// 		if err != nil {
// 			return err
// 		}

// 		it := nd.MapIterator()
// 		for {
// 			if it.Done() {
// 				break
// 			}

// 			k, v, err := it.Next()
// 			if err != nil {
// 				return err
// 			}

// 			err = doBind(ma.AssembleKey(), k)
// 			if err != nil {
// 				return err
// 			}

// 			err = doBind(ma.AssembleValue(), v)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		ma.Finish()
// 	case datamodel.Kind_Int:
// 		fmt.Println("FOUND INT")
// 		v, err := nd.AsInt()
// 		if err != nil {
// 			return err
// 		}
// 		err = nb.AssignInt(v)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

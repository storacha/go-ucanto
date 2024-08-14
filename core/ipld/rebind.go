package ipld

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

// Rebind takes a Node and binds it to the Go type according to the passed schema.
func Rebind(nd datamodel.Node, ptrVal any, typ schema.Type) (rnd datamodel.Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			if asStr, ok := r.(string); ok {
				err = errors.New(asStr)
			} else if asErr, ok := r.(error); ok {
				err = asErr
			} else {
				err = errors.New("unknown panic rebinding node")
			}
		}
	}()

	err = verifyCompatibility(nd, typ)
	if err != nil {
		return
	}

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
	rnd = bindnode.Wrap(ptrVal, typ)
	return
}

// verifyCompatibility checks the node tree matches the schema.
func verifyCompatibility(node datamodel.Node, schemaType schema.Type) error {
	doError := func(format string, args ...interface{}) error {
		errFormat := "rebind: schema type %s is not compatible with node type %s"
		errArgs := []interface{}{schemaType.Name(), node.Kind().String()}
		if format != "" {
			errFormat += ": " + format
		}
		errArgs = append(errArgs, args...)
		return fmt.Errorf(errFormat, errArgs...)
	}
	switch schemaType := schemaType.(type) {
	case *schema.TypeBool:
		if node.Kind() != datamodel.Kind_Bool {
			return doError("kind mismatch; need boolean")
		}
	case *schema.TypeInt:
		if node.Kind() != datamodel.Kind_Int {
			return doError("kind mismatch; need integer")
		}
	case *schema.TypeFloat:
		if node.Kind() != datamodel.Kind_Float {
			return doError("kind mismatch; need float")
		}
	case *schema.TypeString:
		if node.Kind() != datamodel.Kind_String {
			return doError("kind mismatch; need string")
		}
	case *schema.TypeBytes:
		if node.Kind() != datamodel.Kind_Bytes {
			return doError("kind mismatch; need bytes")
		}
	case *schema.TypeEnum:
		if _, ok := schemaType.RepresentationStrategy().(schema.EnumRepresentation_Int); ok {
			if node.Kind() != datamodel.Kind_Int {
				return doError("kind mismatch; need integer enum")
			}
		} else if node.Kind() != datamodel.Kind_String {
			return doError("kind mismatch; need string enum")
		}
	case *schema.TypeList:
		if node.Kind() != datamodel.Kind_List {
			return doError("kind mismatch; need list")
		}

		it := node.ListIterator()
		for {
			if it.Done() {
				break
			}
			_, nd, err := it.Next()
			if err != nil {
				return err
			}
			if nd.Kind() == datamodel.Kind_Null && schemaType.ValueIsNullable() {
				continue
			}

			err = verifyCompatibility(nd, schemaType.ValueType())
			if err != nil {
				return err
			}
		}
	case *schema.TypeMap:
		if node.Kind() != datamodel.Kind_Map {
			return doError("kind mismatch; need map")
		}

		it := node.MapIterator()
		for {
			if it.Done() {
				break
			}
			k, v, err := it.Next()
			if err != nil {
				return err
			}

			err = verifyCompatibility(k, schemaType.KeyType())
			if err != nil {
				return err
			}

			if v.Kind() == datamodel.Kind_Null && schemaType.ValueIsNullable() {
				continue
			}
			err = verifyCompatibility(v, schemaType.ValueType())
			if err != nil {
				return err
			}
		}
	case *schema.TypeStruct:
		if node.Kind() != datamodel.Kind_Map {
			return doError("kind mismatch; need struct")
		}

		for _, schemaField := range schemaType.Fields() {
			schemaType := schemaField.Type()
			vnode, err := node.LookupByString(schemaField.Name())
			if err != nil {
				return err
			}

			switch {
			case schemaField.IsOptional() && schemaField.IsNullable():
				if vnode == nil || vnode.Kind() == datamodel.Kind_Null {
					continue
				}
			case schemaField.IsOptional():
				if vnode == nil {
					continue
				}
				if vnode.Kind() == datamodel.Kind_Null {
					return doError("optional field is not nullable")
				}
			case schemaField.IsNullable():
				if vnode == nil {
					return doError("nullable field is not optional")
				}
				if vnode.Kind() == datamodel.Kind_Null {
					continue
				}
			}
			err = verifyCompatibility(vnode, schemaType)
			if err != nil {
				return err
			}
		}
	case *schema.TypeUnion:
		schemaMembers := schemaType.Members()
		var err error
		for _, schemaType := range schemaMembers {
			err = verifyCompatibility(node, schemaType)
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
	case *schema.TypeLink:
		if node.Kind() != datamodel.Kind_Link {
			return doError("kind mismatch; need link")
		}
	case *schema.TypeAny:
	default:
		return doError(fmt.Sprintf("unexpected schema type: %T", schemaType))
	}
	return nil
}

package schema

import "github.com/ipld/go-ipld-prime/schema"

type strukt struct {
	bind any
	typ  schema.Type
}

func Struct(bind any, typ schema.Type) {

}

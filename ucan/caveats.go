package ucan

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
)

// NoCaveats can be used when a capability has no additional domain specific
// details and/or restrictions.
type NoCaveats struct{}

func (c NoCaveats) Build() (datamodel.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, err := nb.BeginMap(0)
	if err != nil {
		return nil, err
	}
	ma.Finish()
	return nb.Build(), nil
}

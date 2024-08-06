package receipt

import (
	// for go:embed
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/web3-storage/go-ucanto/core/dag/blockstore"
	"github.com/web3-storage/go-ucanto/core/delegation"
	"github.com/web3-storage/go-ucanto/core/invocation"
	"github.com/web3-storage/go-ucanto/core/invocation/ran"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/core/ipld/block"
	"github.com/web3-storage/go-ucanto/core/ipld/codec/cbor"
	"github.com/web3-storage/go-ucanto/core/ipld/hash/sha256"
	"github.com/web3-storage/go-ucanto/core/iterable"
	rdm "github.com/web3-storage/go-ucanto/core/receipt/datamodel"
	"github.com/web3-storage/go-ucanto/core/result"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/ucan"
	"github.com/web3-storage/go-ucanto/ucan/crypto/signature"
)

type Effects interface {
	Fork() []ipld.Link
	Join() ipld.Link
}

// Receipt represents a view of the invocation receipt. This interface provides
// an ergonomic API and allows you to reference linked IPLD objects if they are
// included in the source DAG.
type Receipt[O, X any] interface {
	ipld.View
	Ran() invocation.Invocation
	Out() result.Result[O, X]
	Fx() Effects
	Meta() map[string]any
	Issuer() ucan.Principal
	Proofs() delegation.Proofs
	Signature() signature.SignatureView
}

type results[O, X any] struct {
	model *rdm.ResultModel[O, X]
}

func (r results[O, X]) Error() (X, bool) {
	if r.model.Err != nil {
		return *r.model.Err, true
	}
	var x X
	return x, false
}

func (r results[O, X]) Ok() (O, bool) {
	if r.model.Ok != nil {
		return *r.model.Ok, true
	}
	var o O
	return o, false
}

type effects struct {
	model rdm.EffectsModel
}

func (fx effects) Fork() []ipld.Link {
	return fx.model.Fork
}

func (fx effects) Join() ipld.Link {
	return fx.model.Join
}

type receipt[O, X any] struct {
	rt   block.Block
	blks blockstore.BlockReader
	data *rdm.ReceiptModel[O, X]
}

var _ Receipt[any, any] = (*receipt[any, any])(nil)

func (r *receipt[O, X]) Blocks() iterable.Iterator[block.Block] {
	var iterators []iterable.Iterator[block.Block]
	iterators = append(iterators, r.Ran().Blocks())

	for _, prf := range r.Proofs() {
		if delegation, ok := prf.Delegation(); ok {
			iterators = append(iterators, delegation.Blocks())
		}
	}

	iterators = append(iterators, iterable.From([]block.Block{r.Root()}))

	return iterable.Concat(iterators...)
}

func (r *receipt[O, X]) Fx() Effects {
	return effects{r.data.Ocm.Fx}
}

func (r *receipt[O, X]) Issuer() ucan.Principal {
	if r.data.Ocm.Iss == nil {
		return nil
	}
	principal, err := did.Parse(*r.data.Ocm.Iss)
	if err != nil {
		fmt.Printf("Error: decoding issuer DID: %s\n", err)
	}
	return principal
}

func (r *receipt[O, X]) Proofs() delegation.Proofs {
	return delegation.NewProofsView(r.data.Ocm.Prf, r.blks)
}

// Map values are datamodel.Node
func (r *receipt[O, X]) Meta() map[string]any {
	meta := map[string]any{}
	for k, v := range r.data.Ocm.Meta.Values {
		meta[k] = any(v)
	}
	return meta
}

func (r *receipt[O, X]) Out() result.Result[O, X] {
	return results[O, X]{&r.data.Ocm.Out}
}

func (r *receipt[O, X]) Ran() invocation.Invocation {
	inv, err := invocation.NewInvocationView(r.data.Ocm.Ran, r.blks)
	if err != nil {
		fmt.Printf("Error: creating invocation view: %s\n", err)
	}
	return inv
}

func (r *receipt[O, X]) Root() block.Block {
	return r.rt
}

func (r *receipt[O, X]) Signature() signature.SignatureView {
	return signature.NewSignatureView(signature.Decode(r.data.Sig))
}

func NewReceipt[O, X any](root ipld.Link, blocks blockstore.BlockReader, typ schema.Type) (Receipt[O, X], error) {
	rblock, ok, err := blocks.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting receipt root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing receipt root block: %s", root)
	}

	rmdl := rdm.ReceiptModel[O, X]{}
	err = block.Decode(rblock, &rmdl, typ, cbor.Codec, sha256.Hasher)
	if err != nil {
		return nil, fmt.Errorf("decoding receipt: %s", err)
	}

	rcpt := receipt[O, X]{
		rt:   rblock,
		blks: blocks,
		data: &rmdl,
	}

	return &rcpt, nil
}

type ReceiptReader[O, X any] interface {
	Read(rcpt ipld.Link, blks iterable.Iterator[block.Block]) (Receipt[O, X], error)
}

type receiptReader[O, X any] struct {
	typ schema.Type
}

func (rr *receiptReader[O, X]) Read(rcpt ipld.Link, blks iterable.Iterator[block.Block]) (Receipt[O, X], error) {
	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blks))
	if err != nil {
		return nil, fmt.Errorf("creating block reader: %s", err)
	}
	return NewReceipt[O, X](rcpt, br, rr.typ)
}

func NewReceiptReader[O, X any](resultschema []byte) (ReceiptReader[O, X], error) {
	typ, err := rdm.NewReceiptModelType(resultschema)
	if err != nil {
		return nil, fmt.Errorf("loading receipt data model: %s", err)
	}
	return &receiptReader[O, X]{typ}, nil
}

type AnyReceipt Receipt[datamodel.Node, datamodel.Node]

var (
	anyReceiptTs *schema.TypeSystem
)

//go:embed anyresult.ipldsch
var anyResultSchema []byte

func init() {
	ts, err := rdm.NewReceiptModelType(anyResultSchema)
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	anyReceiptTs = ts.TypeSystem()
}

// Option is an option configuring a UCAN delegation.
type Option func(cfg *receiptConfig) error

type receiptConfig struct {
	meta  map[string]any
	prf   delegation.Proofs
	forks []ipld.Link
	join  ipld.Link
}

// WithProofs configures the proofs for the receipt. If the `issuer` of this
// `Receipt` is not the resource owner / service provider, for the delegated
// capabilities, the `proofs` must contain valid `Proof`s containing
// delegations to the `issuer`.
func WithProofs(prf delegation.Proofs) Option {
	return func(cfg *receiptConfig) error {
		cfg.prf = prf
		return nil
	}
}

// WithMeta configures the metadata for the receipt.
func WithMeta(meta map[string]any) Option {
	return func(cfg *receiptConfig) error {
		cfg.meta = meta
		return nil
	}
}

// WithForks configures the forks for the receipt.
func WithForks(forks []ipld.Link) Option {
	return func(cfg *receiptConfig) error {
		cfg.forks = forks
		return nil
	}
}

// WithJoin configures the join for the receipt.
func WithJoin(join ipld.Link) Option {
	return func(cfg *receiptConfig) error {
		cfg.join = join
		return nil
	}
}

func wrapOrFail(value interface{}) (nd schema.TypedNode, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	nd = bindnode.Wrap(value, nil)
	return
}

func Issue[O, X ipld.Node](issuer ucan.Signer, result result.Result[O, X], ran ran.Ran, opts ...Option) (Receipt[O, X], error) {
	cfg := receiptConfig{}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	bs, err := blockstore.NewBlockStore()
	if err != nil {
		return nil, err
	}

	// copy invocation blocks into the store
	invocationLink, err := ran.WriteInto(bs)
	if err != nil {
		return nil, err
	}

	// copy proof blocks into store
	prooflinks, err := cfg.prf.WriteInto(bs)
	if err != nil {
		return nil, err
	}

	effectsModel := rdm.EffectsModel{
		Fork: cfg.forks,
		Join: cfg.join,
	}

	metaModel := rdm.MetaModel{}
	// attempt to convert meta into IPLD format if present.
	if cfg.meta != nil {
		metaModel.Values = make(map[string]datamodel.Node, len(cfg.meta))
		for k, v := range cfg.meta {
			nd, err := wrapOrFail(v)
			if err != nil {
				return nil, err
			}
			metaModel.Keys = append(metaModel.Keys, k)
			metaModel.Values[k] = nd
		}
	}

	resultModel := rdm.ResultModel[O, X]{}
	if success, ok := result.Ok(); ok {
		resultModel.Ok = &success
		fmt.Println("success kind:")
		fmt.Println(success.Kind().String())
	}
	if err, ok := result.Error(); ok {
		resultModel.Err = &err
		fmt.Println("error kind:")
		fmt.Println(err.Kind().String())
	}

	issString := issuer.DID().String()
	outcomeModel := rdm.OutcomeModel[O, X]{
		Ran:  invocationLink,
		Out:  resultModel,
		Fx:   effectsModel,
		Iss:  &issString,
		Meta: metaModel,
		Prf:  prooflinks,
	}

	outcomeBytes, err := cbor.Encode(&outcomeModel, anyReceiptTs.TypeByName("Outcome"))
	if err != nil {
		return nil, err
	}
	signature := issuer.Sign(outcomeBytes).Bytes()

	receiptModel := rdm.ReceiptModel[O, X]{
		Ocm: outcomeModel,
		Sig: signature,
	}

	rt, err := block.Encode(receiptModel, anyReceiptTs.TypeByName("Receipt"), cbor.Codec, sha256.Hasher)
	if err != nil {
		return nil, fmt.Errorf("encoding receipt: %s", err)
	}

	err = bs.Put(rt)
	if err != nil {
		return nil, fmt.Errorf("adding receipt root to store: %s", err)
	}

	return &receipt[O, X]{
		rt:   rt,
		blks: bs,
		data: &receiptModel,
	}, nil
}

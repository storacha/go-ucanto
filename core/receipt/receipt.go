package receipt

import (
	// for go:embed
	_ "embed"
	"fmt"
	"iter"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/invocation/ran"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	"github.com/storacha/go-ucanto/core/iterable"
	rdm "github.com/storacha/go-ucanto/core/receipt/datamodel"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

// Receipt represents a view of the invocation receipt. This interface provides
// an ergonomic API and allows you to reference linked IPLD objects if they are
// included in the source DAG.
type Receipt[O, X any] interface {
	ipld.View
	Ran() invocation.Invocation
	Out() result.Result[O, X]
	Fx() fx.Effects
	Meta() map[string]any
	Issuer() ucan.Principal
	Proofs() delegation.Proofs
	Signature() signature.SignatureView
}

func toResultModel[O, X any](res result.Result[O, X]) rdm.ResultModel[O, X] {
	return result.MatchResultR1(res, func(ok O) rdm.ResultModel[O, X] {
		return rdm.ResultModel[O, X]{Ok: &ok, Err: nil}
	}, func(err X) rdm.ResultModel[O, X] {
		return rdm.ResultModel[O, X]{Ok: nil, Err: &err}
	})
}

func fromResultModel[O, X any](resultModel rdm.ResultModel[O, X]) result.Result[O, X] {
	if resultModel.Ok != nil {
		return result.Ok[O, X](*resultModel.Ok)
	}
	return result.Error[O, X](*resultModel.Err)
}

type receipt[O, X any] struct {
	rt   block.Block
	blks blockstore.BlockReader
	data *rdm.ReceiptModel[O, X]
}

var _ Receipt[any, any] = (*receipt[any, any])(nil)

func (r *receipt[O, X]) Blocks() iter.Seq2[block.Block, error] {
	var iterators []iter.Seq2[block.Block, error]
	iterators = append(iterators, r.Ran().Blocks())

	for _, prf := range r.Proofs() {
		if delegation, ok := prf.Delegation(); ok {
			iterators = append(iterators, delegation.Blocks())
		}
	}

	iterators = append(iterators, func(yield func(block.Block, error) bool) { yield(r.Root(), nil) })

	return iterable.Concat2(iterators...)
}

func (r *receipt[O, X]) Fx() fx.Effects {
	var fork []fx.Effect
	var join fx.Effect
	for _, l := range r.data.Ocm.Fx.Fork {
		b, _, _ := r.blks.Get(l)
		if b != nil {
			inv, _ := delegation.NewDelegation(b, r.blks)
			fork = append(fork, fx.FromInvocation(inv))
		} else {
			fork = append(fork, fx.FromLink(l))
		}
	}

	if r.data.Ocm.Fx.Join != nil {
		b, _, _ := r.blks.Get(r.data.Ocm.Fx.Join)
		if b != nil {
			inv, _ := delegation.NewDelegation(b, r.blks)
			join = fx.FromInvocation(inv)
		} else {
			join = fx.FromLink(r.data.Ocm.Fx.Join)
		}
	}

	return fx.NewEffects(fx.WithFork(fork...), fx.WithJoin(join))
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
	return fromResultModel(r.data.Ocm.Out)
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
	Read(rcpt ipld.Link, blks iter.Seq2[block.Block, error]) (Receipt[O, X], error)
}

type receiptReader[O, X any] struct {
	typ schema.Type
}

func (rr *receiptReader[O, X]) Read(rcpt ipld.Link, blks iter.Seq2[block.Block, error]) (Receipt[O, X], error) {
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

func NewReceiptReaderFromTypes[O, X any](successType schema.Type, errType schema.Type) (ReceiptReader[O, X], error) {
	typ, err := rdm.NewReceiptModelFromTypes(successType, errType)
	if err != nil {
		return nil, fmt.Errorf("loading receipt data model: %s", err)
	}
	return &receiptReader[O, X]{typ}, nil
}

type AnyReceipt Receipt[ipld.Node, ipld.Node]

// Option is an option configuring a UCAN delegation.
type Option func(cfg *receiptConfig) error

type receiptConfig struct {
	meta  map[string]any
	prf   delegation.Proofs
	forks []fx.Effect
	join  fx.Effect
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

// WithFork configures the forks for the receipt.
func WithFork(forks ...fx.Effect) Option {
	return func(cfg *receiptConfig) error {
		cfg.forks = forks
		return nil
	}
}

// WithJoin configures the join for the receipt.
func WithJoin(join fx.Effect) Option {
	return func(cfg *receiptConfig) error {
		cfg.join = join
		return nil
	}
}

func Issue[O, X ipld.Builder](issuer ucan.Signer, out result.Result[O, X], ran ran.Ran, opts ...Option) (AnyReceipt, error) {
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

	var forks []ipld.Link
	for _, effect := range cfg.forks {
		if inv, ok := effect.Invocation(); ok {
			blockstore.WriteInto(inv, bs)
		}
		forks = append(forks, effect.Link())
	}

	var join ipld.Link
	if cfg.join != (fx.Effect{}) {
		if inv, ok := cfg.join.Invocation(); ok {
			blockstore.WriteInto(inv, bs)
		}
		join = cfg.join.Link()
	}

	effectsModel := rdm.EffectsModel{
		Fork: forks,
		Join: join,
	}

	metaModel := rdm.MetaModel{}
	// attempt to convert meta into IPLD format if present.
	if cfg.meta != nil {
		metaModel.Values = make(map[string]datamodel.Node, len(cfg.meta))
		for k, v := range cfg.meta {
			nd, err := ipld.WrapWithRecovery(v, nil)
			if err != nil {
				return nil, err
			}
			metaModel.Keys = append(metaModel.Keys, k)
			metaModel.Values[k] = nd
		}
	}

	nodeResult, err := result.MapResultR1(out, func(b O) (ipld.Node, error) {
		return b.ToIPLD()
	}, func(b X) (ipld.Node, error) {
		return b.ToIPLD()
	})
	if err != nil {
		return nil, err
	}
	resultModel := toResultModel(nodeResult)
	issString := issuer.DID().String()
	outcomeModel := rdm.OutcomeModel[ipld.Node, ipld.Node]{
		Ran:  invocationLink,
		Out:  resultModel,
		Fx:   effectsModel,
		Iss:  &issString,
		Meta: metaModel,
		Prf:  prooflinks,
	}

	outcomeBytes, err := cbor.Encode(&outcomeModel, rdm.TypeSystem().TypeByName("Outcome"))
	if err != nil {
		return nil, err
	}
	signature := issuer.Sign(outcomeBytes).Bytes()

	receiptModel := rdm.ReceiptModel[ipld.Node, ipld.Node]{
		Ocm: outcomeModel,
		Sig: signature,
	}

	rt, err := block.Encode(&receiptModel, rdm.TypeSystem().TypeByName("Receipt"), cbor.Codec, sha256.Hasher)
	if err != nil {
		return nil, fmt.Errorf("encoding receipt: %s", err)
	}

	err = bs.Put(rt)
	if err != nil {
		return nil, fmt.Errorf("adding receipt root to store: %s", err)
	}

	return &receipt[ipld.Node, ipld.Node]{
		rt:   rt,
		blks: bs,
		data: &receiptModel,
	}, nil
}

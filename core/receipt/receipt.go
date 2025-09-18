package receipt

import (
	// for go:embed

	"bytes"
	_ "embed"
	"fmt"
	"io"
	"iter"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	"github.com/storacha/go-ucanto/core/iterable"
	rdm "github.com/storacha/go-ucanto/core/receipt/datamodel"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/receipt/ran"
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
	Ran() ran.Ran
	Out() result.Result[O, X]
	Fx() fx.Effects
	Meta() map[string]any
	Issuer() ucan.Principal
	Proofs() delegation.Proofs
	Signature() signature.SignatureView
	Archive() io.Reader
	Export() iter.Seq2[block.Block, error]
}

func toResultModel[O, X any](res result.Result[O, X]) rdm.ResultModel[O, X] {
	return result.MatchResultR1(res, func(ok O) rdm.ResultModel[O, X] {
		return rdm.ResultModel[O, X]{Ok: &ok, Error: nil}
	}, func(err X) rdm.ResultModel[O, X] {
		return rdm.ResultModel[O, X]{Ok: nil, Error: &err}
	})
}

func fromResultModel[O, X any](resultModel rdm.ResultModel[O, X]) result.Result[O, X] {
	if resultModel.Ok != nil {
		return result.Ok[O, X](*resultModel.Ok)
	}
	return result.Error[O, X](*resultModel.Error)
}

var _ Receipt[any, any] = (*receipt[any, any])(nil)

type receipt[O, X any] struct {
	rt   block.Block
	blks blockstore.BlockReader
	data *rdm.ReceiptModel[O, X]
}

func NewReceipt[O, X any](root ipld.Link, blocks blockstore.BlockReader, typ schema.Type, opts ...bindnode.Option) (Receipt[O, X], error) {
	rblock, ok, err := blocks.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting receipt root block: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing receipt root block: %s", root)
	}

	rmdl := rdm.ReceiptModel[O, X]{}
	err = block.Decode(rblock, &rmdl, typ, cbor.Codec, sha256.Hasher, opts...)
	if err != nil {
		return nil, fmt.Errorf("decoding receipt: %w", err)
	}

	rcpt := receipt[O, X]{
		rt:   rblock,
		blks: blocks,
		data: &rmdl,
	}

	return &rcpt, nil
}

func NewAnyReceipt(root ipld.Link, blocks blockstore.BlockReader, opts ...bindnode.Option) (AnyReceipt, error) {
	anyReceiptType := rdm.TypeSystem().TypeByName("Receipt")
	return NewReceipt[ipld.Node, ipld.Node](root, blocks, anyReceiptType, opts...)
}

func (r *receipt[O, X]) Blocks() iter.Seq2[block.Block, error] {
	return r.blks.Iterator()
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

func (r *receipt[O, X]) Ran() ran.Ran {
	_, ok, err := r.blks.Get(r.data.Ocm.Ran)
	if !ok || err != nil {
		return ran.FromLink(r.data.Ocm.Ran)
	}
	inv, err := invocation.NewInvocationView(r.data.Ocm.Ran, r.blks)
	if err != nil {
		fmt.Printf("Error: creating invocation view: %s\n", err)
		return ran.FromLink(r.data.Ocm.Ran)
	}
	return ran.FromInvocation(inv)
}

func (r *receipt[O, X]) Root() block.Block {
	return r.rt
}

func (r *receipt[O, X]) Signature() signature.SignatureView {
	return signature.NewSignatureView(signature.Decode(r.data.Sig))
}

func (r *receipt[O, X]) Archive() io.Reader {
	// We create a descriptor block to describe what this DAG represents
	variant, err := block.Encode(
		&rdm.ArchiveModel{UcanReceipt0_9_1: r.rt.Link()},
		rdm.ArchiveType(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		reader, _ := io.Pipe()
		reader.CloseWithError(fmt.Errorf("hashing variant block bytes: %w", err))
		return reader
	}

	return car.Encode([]ipld.Link{variant.Link()}, func(yield func(ipld.Block, error) bool) {
		for b, err := range r.Export() {
			if !yield(b, err) || err != nil {
				return
			}
		}
		yield(variant, nil)
	})
}

// Export ONLY the blocks that comprise the receipt, its original invocation and its proofs
// This differs from Blocks(), which simply returns all the blocks in the backing blockstore
func (r *receipt[O, X]) Export() iter.Seq2[block.Block, error] {
	var iterators []iter.Seq2[block.Block, error]

	if inv, ok := r.Ran().Invocation(); ok {
		iterators = append(iterators, inv.Export())
	}

	for _, prf := range r.Proofs() {
		if delegation, ok := prf.Delegation(); ok {
			iterators = append(iterators, delegation.Export())
		}
	}

	iterators = append(iterators, func(yield func(block.Block, error) bool) { yield(r.Root(), nil) })

	return iterable.Concat2(iterators...)
}

type ReceiptReader[O, X any] interface {
	Read(rcpt ipld.Link, blks iter.Seq2[block.Block, error]) (Receipt[O, X], error)
}

type receiptReader[O, X any] struct {
	typ  schema.Type
	opts []bindnode.Option
}

func (rr *receiptReader[O, X]) Read(rcpt ipld.Link, blks iter.Seq2[block.Block, error]) (Receipt[O, X], error) {
	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blks))
	if err != nil {
		return nil, fmt.Errorf("creating block reader: %w", err)
	}
	return NewReceipt[O, X](rcpt, br, rr.typ, rr.opts...)
}

func NewReceiptReader[O, X any](resultschema []byte, opts ...bindnode.Option) (ReceiptReader[O, X], error) {
	typ, err := rdm.NewReceiptModelType(resultschema)
	if err != nil {
		return nil, fmt.Errorf("loading receipt data model: %w", err)
	}
	return &receiptReader[O, X]{typ, opts}, nil
}

func NewAnyReceiptReader(opts ...bindnode.Option) ReceiptReader[ipld.Node, ipld.Node] {
	anyReceiptType := rdm.TypeSystem().TypeByName("Receipt")
	return &receiptReader[ipld.Node, ipld.Node]{anyReceiptType, opts}
}

func NewReceiptReaderFromTypes[O, X any](successType schema.Type, errType schema.Type, opts ...bindnode.Option) (ReceiptReader[O, X], error) {
	typ, err := rdm.NewReceiptModelFromTypes(successType, errType)
	if err != nil {
		return nil, fmt.Errorf("loading receipt data model: %w", err)
	}
	return &receiptReader[O, X]{typ, opts}, nil
}

type AnyReceipt Receipt[ipld.Node, ipld.Node]

func Rebind[O, X any](from AnyReceipt, successType schema.Type, errorType schema.Type, opts ...bindnode.Option) (Receipt[O, X], error) {
	rdr, err := NewReceiptReaderFromTypes[O, X](successType, errorType, opts...)
	if err != nil {
		return nil, err
	}
	return rdr.Read(from.Root().Link(), from.Blocks())
}

func Extract(b []byte) (AnyReceipt, error) {
	roots, blks, err := car.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("decoding CAR: %s", err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("missing root CID in receipt archive")
	}
	if len(roots) > 1 {
		return nil, fmt.Errorf("unexpected number of root CIDs in archive: %d", len(roots))
	}

	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blks))
	if err != nil {
		return nil, fmt.Errorf("creating block reader: %w", err)
	}

	rt, ok, err := br.Get(roots[0])
	if err != nil {
		return nil, fmt.Errorf("getting root block: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing root block: %s", roots[0])
	}

	model := rdm.ArchiveModel{}
	err = block.Decode(
		rt,
		&model,
		rdm.ArchiveType(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		return nil, fmt.Errorf("decoding root block: %w", err)
	}

	return NewAnyReceipt(model.UcanReceipt0_9_1, br)
}

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
		return nil, fmt.Errorf("encoding receipt: %w", err)
	}

	err = bs.Put(rt)
	if err != nil {
		return nil, fmt.Errorf("adding receipt root to store: %w", err)
	}

	return &receipt[ipld.Node, ipld.Node]{
		rt:   rt,
		blks: bs,
		data: &receiptModel,
	}, nil
}

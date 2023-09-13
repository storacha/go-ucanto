package car

import (
	"fmt"
	"io"

	coreipld "github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
	"github.com/ipfs/go-car/util"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/ipld/go-ipld-prime"
)

// ContentType is the value the HTTP Content-Type header should have for CARs.
// See https://www.iana.org/assignments/media-types/application/vnd.ipld.car
const ContentType = "application/vnd.ipld.car"

func init() {
	cbor.RegisterCborType(CarHeader{})
}

type CarHeader struct {
	Roots   []ipld.Link
	Version uint64
}

func Encode(roots []ipld.Link, blocks iterable.Iterator[coreipld.Block]) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		h := CarHeader{}
		hb, err := cbor.DumpObject(h)
		if err != nil {
			writer.CloseWithError(fmt.Errorf("writing CAR header: %s", err))
			return
		}
		util.LdWrite(writer, hb)
		for {
			block, err := blocks.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				writer.CloseWithError(fmt.Errorf("writing CAR blocks: %s", err))
				return
			}
			util.LdWrite(writer, []byte(block.Link().Binary()), block.Bytes())
		}
		writer.Close()
	}()
	return reader
}

package car

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

type fixture struct {
	path   string
	root   ipld.Link
	blocks []ipld.Link
}

var fixtures = []fixture{
	{
		path: "testdata/lost-dog.jpg.car",
		root: cidlink.Link{Cid: cid.MustParse("bafybeif4owy5gno5lwnixqm52rwqfodklf76hsetxdhffuxnplvijskzqq")},
		blocks: []ipld.Link{
			cidlink.Link{Cid: cid.MustParse("bafkreifau35r7vi37tvbvfy3hdwvgb4tlflqf7zcdzeujqcjk3rsphiwte")},
			cidlink.Link{Cid: cid.MustParse("bafkreicj3ozpzd46nx26hflpoi6hgm5linwo65cvphd5ol3ke3vk5nb7aa")},
			cidlink.Link{Cid: cid.MustParse("bafybeihkqv2ukwgpgzkwsuz7whmvneztvxglkljbs3zosewgku2cfluvba")},
			cidlink.Link{Cid: cid.MustParse("bafybeif4owy5gno5lwnixqm52rwqfodklf76hsetxdhffuxnplvijskzqq")},
		},
	},
}

func TestDecodeCAR(t *testing.T) {
	file, err := os.Open(fixtures[0].path)
	if err != nil {
		t.Fatal(err)
	}

	roots, blocks, err := Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("unexpected number of roots: %d, expected: 1", len(roots))
	}
	if roots[0].String() != fixtures[0].root.String() {
		t.Fatalf("unexpected root: %s, expected: %s", roots[0], fixtures[0].root)
	}

	var blks []ipld.Block
	for {
		b, err := blocks.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("reading blocks: %s", err)
		}
		blks = append(blks, b)
	}

	if len(blks) != len(fixtures[0].blocks) {
		t.Fatalf("incorrect number of blocks: %d, expected: %d", len(blks), len(fixtures[0].blocks))
	}
	for i, b := range fixtures[0].blocks {
		if b.String() != blks[i].Link().String() {
			t.Fatalf("unexpected block: %s, expected: %s", b, blks[i].Link())
		}
	}
}

func TestEncodeCAR(t *testing.T) {
	file, err := os.Open(fixtures[0].path)
	if err != nil {
		t.Fatal(err)
	}

	fbytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	roots, blocks, err := Decode(bytes.NewReader(fbytes))
	if err != nil {
		t.Fatal(err)
	}

	rd := Encode(roots, blocks)

	dbytes, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(fbytes, dbytes) {
		t.Fatal("failed to round trip")
	}
}

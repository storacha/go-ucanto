package printer

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ipld/go-ipld-prime/printer"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/stretchr/testify/require"
)

func withIndent(t *testing.T, level int) func(format string, args ...any) {
	indent := strings.Repeat("  ", level)
	return func(format string, args ...any) {
		t.Logf(indent+format, args...)
	}
}

func PrintDelegation(t *testing.T, d delegation.Delegation, level int) {
	t.Helper()
	log := withIndent(t, level)

	log("%s\n", d.Link())
	log("  Issuer: %s", d.Issuer().DID())
	log("  Audience: %s", d.Audience().DID())

	log("  Capabilities:")
	for _, c := range d.Capabilities() {
		log("    Can: %s", c.Can())
		log("    With: %s", c.With())
		log("    Nb: %v", c.Nb())
	}

	if d.Expiration() != nil {
		log("  Expiration: %s", time.Unix(int64(*d.Expiration()), 0).String())
	}

	bs, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(d.Blocks()))
	require.NoError(t, err)

	if len(d.Proofs()) > 0 {
		log("  Proofs:")
		for _, p := range d.Proofs() {
			pd, err := delegation.NewDelegationView(p, bs)
			if err != nil {
				log("    %s\n", p)
				continue
			}
			PrintDelegation(t, pd, level+2)
		}
	}
}

func PrintReceipt(t *testing.T, r receipt.AnyReceipt) {
	t.Helper()
	t.Logf("%s", r.Root().Link())
	t.Logf("  Issuer: %s", r.Issuer().DID())
	inv, ok := r.Ran().Invocation()
	if ok {
		t.Logf("  Ran:")
		PrintDelegation(t, inv, 2)
	} else {
		t.Logf("  Ran: %s", r.Ran().Link())
	}
	t.Log("  Out:")
	o, x := result.Unwrap(r.Out())
	if x != nil {
		t.Logf("    Error:\n      %s", printer.Sprint(x))
	} else {
		t.Logf("    OK:\n      %s", printer.Sprint(o))
	}
}

func PrintNode(t *testing.T, n ipld.Node) {
	t.Helper()
	t.Log(printer.Sprint(n))
}

func SprintBytes(t *testing.T, b int) string {
	t.Helper()
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func PrintHeaders(t *testing.T, h http.Header) {
	t.Helper()
	for name, values := range h {
		for _, value := range values {
			t.Logf("%s: %s", name, value)
		}
	}
}

package verifier

import "testing"

func TestParse(t *testing.T) {
	str := "did:key:z6MkgZN5cRgWqesJeaZCEs7eKzyQsfpzmhnSEqTL6FZt56Ym"
	v, err := Parse(str)
	if err != nil {
		t.Fatalf("parsing DID: %s", err)
	}
	if v.DID().String() != str {
		t.Fatalf("expected %s to equal %s", v.DID().String(), str)
	}
}

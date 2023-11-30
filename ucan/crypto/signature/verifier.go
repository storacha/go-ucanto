package signature

// Verifier represents an entity that can verify signatures.
type Verifier interface {
	// Verify takes a byte encoded message and verifies that it is signed by
	// corresponding signer.
	Verify(msg []byte, sig Signature) bool
}

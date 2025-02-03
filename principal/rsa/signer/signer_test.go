package signer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func TestGenerateEncodeDecode(t *testing.T) {
	s0 := helpers.Must(Generate())
	fmt.Println(s0.DID().String())

	s1 := helpers.Must(Decode(s0.Encode()))
	fmt.Println(s1.DID().String())

	require.Equal(t, s0.DID().String(), s1.DID().String())
}

func TestGenerateFormatParse(t *testing.T) {
	s0 := helpers.Must(Generate())
	fmt.Println(s0.DID().String())

	str := helpers.Must(Format(s0))
	fmt.Println(str)

	s1 := helpers.Must(Parse(str))
	fmt.Println(s1.DID().String())

	require.Equal(t, s0.DID().String(), s1.DID().String())
}

func TestVerify(t *testing.T) {
	s0 := helpers.Must(Generate())

	msg := []byte("testy")
	sig := s0.Sign(msg)

	res := s0.Verifier().Verify(msg, sig)
	require.Equal(t, true, res)
}

func TestFromRaw(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, keySize)
	require.NoError(t, err)

	raw := x509.MarshalPKCS1PrivateKey(priv)

	s, err := FromRaw(raw)
	require.NoError(t, err)

	fmt.Println(s.DID())

	require.Equal(t, raw, s.Raw())
}

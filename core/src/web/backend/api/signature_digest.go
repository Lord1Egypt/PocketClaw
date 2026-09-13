package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Signature material must not carry secret values.
//
// The gateway restart decision has to be sensitive to a changed credential --
// that is the whole point of PC-DEF-050 -- but being sensitive to a secret does
// not require retaining it. Every component of the signature that can contain
// credential material is therefore reduced to a digest: the plaintext exists
// only as a local that is hashed and dropped inside the call, and what the
// process holds for the lifetime of the gateway is non-reversible.
//
// This matters because `gateway.bootConfigSignature` is long-lived package
// state. Today nothing logs, persists, serialises or embeds it in an error, but
// a single `logger.Debugf("signature=%s", ...)` added during some future
// debugging session would turn a comparison value into a credential dump. The
// digest removes that possibility rather than relying on nobody doing it.
//
// Truncated to 32 hex characters (128 bits). This is a change detector, not an
// authentication tag: it is compared against a value computed by the same
// process from the same code, so there is no adversary choosing inputs to
// collide, and 128 bits is far beyond what accidental collision needs.
const signatureDigestHexLength = 32

// signatureDigest reduces material to a short non-reversible token. Each part is
// written with a length prefix so that concatenation cannot be ambiguous --
// ("ab", "c") and ("a", "bc") must not produce the same digest.
func signatureDigest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		writeLengthPrefixed(hash, part)
	}
	return hex.EncodeToString(hash.Sum(nil))[:signatureDigestHexLength]
}

// signatureDigestBytes is signatureDigest for one already-encoded blob, such as
// a marshalled settings subtree.
func signatureDigestBytes(material []byte) string {
	sum := sha256.Sum256(material)
	return hex.EncodeToString(sum[:])[:signatureDigestHexLength]
}

func writeLengthPrefixed(w io.Writer, value string) {
	var header [8]byte
	length := uint64(len(value))
	for i := 0; i < 8; i++ {
		header[i] = byte(length >> (8 * (7 - i)))
	}
	_, _ = w.Write(header[:])
	_, _ = io.WriteString(w, value)
}

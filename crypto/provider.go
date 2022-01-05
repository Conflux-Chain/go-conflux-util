package crypto

import (
	"crypto/ecdsa"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
)

// PrivKeyIdentifier unique identifier from which private key will be derived.
type PrivKeyIdentifier string

// PrivKeyProvider private key service provider.
// The private key used is generated from (or selected by) a unique identifier deterministically.
type PrivKeyProvider interface {
	// PublicKey returns the public key.
	PublicKey(kid PrivKeyIdentifier) (*ecdsa.PublicKey, error)

	// Address returns the blockchain address with the specified network ID.
	Address(kid PrivKeyIdentifier, networkID ...uint32) (*cfxaddress.Address, error)

	// SignHex calculates an ECDSA signature from a digest hash hex string.
	// The produced signature is in the [R || S || V] format where V is 0 or 1.
	SignHex(kid PrivKeyIdentifier, digestHash types.Hash) (sig []byte, err error)

	// Sign calculates an ECDSA signature from a digest hash bytes.
	Sign(kid PrivKeyIdentifier, digestHash []byte) (sig []byte, err error)

	// SignTx signs an unsigned transaction to a signed transaction.
	SignTx(kid PrivKeyIdentifier, tx *types.UnsignedTransaction) (signedTx []byte, err error)
}

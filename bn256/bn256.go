// Package bn256 allows to use Handel with the BLS signature scheme over the
// BN256 groups. It implements the relevant Handel interfaces: PublicKey,
// Secretkey and MultiSignature.
package bn256

import "github.com/ConsenSys/handel"
import "io"

// NewKeyPair returns a SecretKey using the BN256 curves
func NewKeyPair(reader io.Reader) handel.SecretKey {
	return nil
}

type publicKey struct {
}

type secretKey struct {
}

type multiSig struct {
}

// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/youchainhq/go-youchain/common"
	"github.com/youchainhq/go-youchain/crypto"
	"github.com/youchainhq/go-youchain/params"
)

var (
	ErrInvalidNetworkId = errors.New("invalid network id for signer")
	ErrNotProtected     = errors.New("not protected from replay attack")
)

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type sigCache struct {
	signer Signer
	from   common.Address
}

// MakeSigner returns a Signer based on the given network config and block number.
func MakeSigner(blockNumber *big.Int) Signer {
	var signer Signer = NewYouSigner(params.NetworkId())
	return signer
}

// SignTx signs the transaction using the given signer and private key
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// Sender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// Signer encapsulates transaction signature handling. Note that this interface is not a
// stable API and may change at any time to accommodate new protocol rules.
type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *Transaction) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

// YouSigner implements Signer.
type YouSigner struct {
	networkId, networkIdMul *big.Int
}

func NewYouSigner(networkId uint64) YouSigner {
	return YouSigner{
		networkId:    new(big.Int).SetUint64(networkId),
		networkIdMul: new(big.Int).Mul(new(big.Int).SetUint64(networkId), big.NewInt(2)),
	}
}

func (s YouSigner) Equal(s2 Signer) bool {
	signer, ok := s2.(YouSigner)
	return ok && signer.networkId.Cmp(s.networkId) == 0
}

var big8 = big.NewInt(8)

func (s YouSigner) Sender(tx *Transaction) (common.Address, error) {
	if !tx.Protected() {
		return common.Address{}, ErrNotProtected
	}
	if tx.NetworkId().Cmp(s.networkId) != 0 {
		return common.Address{}, ErrInvalidNetworkId
	}
	V := new(big.Int).Sub(tx.data.V, s.networkIdMul)
	V.Sub(V, big8)
	return recoverPlain(s.Hash(tx), tx.data.R, tx.data.S, V, true)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s YouSigner) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	if s.networkId.Sign() == 0 {
		return nil, nil, nil, ErrInvalidNetworkId
	}
	R, S, V, err = basicSignatureValues(tx, sig)
	if err != nil {
		return nil, nil, nil, err
	}
	// protect from replaying
	V = big.NewInt(int64(sig[64] + 35))
	V.Add(V, s.networkIdMul)
	return R, S, V, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s YouSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
		s.networkId, uint(0), uint(0),
	})
}

// basicSignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func basicSignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

// deriveNetworkId derives the chain id from the given v parameter
func deriveNetworkId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

// Copyright 2020 YOUCHAIN FOUNDATION LTD.
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

package bind

import (
	"crypto/ecdsa"
	"errors"
	"io"
	"io/ioutil"

	"github.com/youchainhq/go-youchain/accounts"
	"github.com/youchainhq/go-youchain/accounts/keystore"
	"github.com/youchainhq/go-youchain/common"
	"github.com/youchainhq/go-youchain/core/types"
	"github.com/youchainhq/go-youchain/crypto"
)

// NewTransactor is a utility method to easily create a transaction signer from
// an encrypted json key stream and the associated passphrase.
func NewTransactor(keyin io.Reader, passphrase string, networkID uint64) (*TransactOpts, error) {
	if networkID == 0 {
		return nil, types.ErrInvalidNetworkId
	}
	json, err := ioutil.ReadAll(keyin)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactor(key.PrivateKey, networkID), nil
}

// NewKeyStoreTransactor is a utility method to easily create a transaction signer from
// an decrypted key from a keystore
func NewKeyStoreTransactor(accMgr accounts.AccountManager, account accounts.Account, networkID uint64) (*TransactOpts, error) {
	if networkID == 0 {
		return nil, types.ErrInvalidNetworkId
	}
	return &TransactOpts{
		From: account.Address,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != account.Address {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := accMgr.SignHash(account, signer.Hash(tx).Bytes())
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		NetworkID: networkID,
	}, nil
}

// NewKeyedTransactor is a utility method to easily create a transaction signer
// from a single private key.
func NewKeyedTransactor(key *ecdsa.PrivateKey, networkID uint64) *TransactOpts {
	if networkID == 0 {
		return nil // It's the only error case, the caller will known this when it get a nil
	}
	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	return &TransactOpts{
		From: keyAddr,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		NetworkID: networkID,
	}
}
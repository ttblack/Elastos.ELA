// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RevertToDPOSTransaction struct {
	BaseTransaction
}


func (t *RevertToDPOSTransaction) CheckTxHeightVersion() error {
	if t.contextParameters.BlockHeight < t.contextParameters.Config.RevertToPOWStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before RevertToPOWStartHeight", t.TxType().Name()))
	}

	return nil
}

func (t *RevertToDPOSTransaction) SpecialCheck() (elaerr.ELAError, bool) {
	p, ok := t.Payload().(*payload.RevertToDPOS)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload.RevertToDPOS")), true
	}
	if p.WorkHeightInterval != payload.WorkHeightInterval {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid WorkHeightInterval")), true

	}

	// check dpos state
	if t.contextParameters.BlockChain.GetState().GetConsensusAlgorithm() != state.POW {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid GetConsensusAlgorithm() != state.POW")), true
	}

	// to avoid init DPOSWorkHeight repeatedly
	if t.contextParameters.BlockChain.GetState().DPOSWorkHeight > t.contextParameters.BlockHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("already receieved  revertodpos")), true
	}

	return elaerr.Simple(elaerr.ErrTxPayload, checkArbitratorsSignatures(t.Programs()[0])), true
}

func checkArbitratorsSignatures(program *program.Program) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1

	var arbitratorsCount int
	arbiters := blockchain.DefaultLedger.Arbitrators.GetArbitrators()
	for _, a := range arbiters {
		if a.IsNormal {
			arbitratorsCount++
		}
	}
	minSignCount := int(float64(blockchain.DefaultLedger.Arbitrators.GetArbitersCount())*
		state.MajoritySignRatioNumerator/state.MajoritySignRatioDenominator) + 1
	if m < 1 || m > n || n != arbitratorsCount || m < minSignCount {
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	for _, pk := range publicKeys {
		if !blockchain.DefaultLedger.Arbitrators.IsArbitrator(pk[1:]) {
			return errors.New("invalid multi sign public key")
		}
	}

	return nil
}

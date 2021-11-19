// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/common"
	"math"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CoinBaseTransaction struct {
	BaseTransaction
}

func (t *CoinBaseTransaction) CheckTransactionInput() error {
	txn := t.sanityParameters.Transaction
	if len(txn.Inputs()) != 1 {
		return errors.New("coinbase must has only one input")
	}
	inputHash := txn.Inputs()[0].Previous.TxID
	inputIndex := txn.Inputs()[0].Previous.Index
	sequence := txn.Inputs()[0].Sequence
	if !inputHash.IsEqual(common.EmptyHash) ||
		inputIndex != math.MaxUint16 || sequence != math.MaxUint32 {
		return errors.New("invalid coinbase input")
	}

	return nil
}

func (t *CoinBaseTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (a *CoinBaseTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	para := a.contextParameters
	if para.BlockHeight >= para.Config.CRCommitteeStartHeight {
		if para.BlockChain.GetState().GetConsensusAlgorithm() == 0x01 {
			if !a.outputs[0].ProgramHash.IsEqual(para.Config.DestroyELAAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be "+
						"DestroyAddress in POW consensus algorithm")), true
			}
		} else {
			if !a.outputs[0].ProgramHash.IsEqual(para.Config.CRAssetsAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be CR assets address")), true
			}
		}
	} else if !a.outputs[0].ProgramHash.IsEqual(para.Config.Foundation) {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("first output address should be foundation address")), true
	}

	return nil, true
}

func (a *CoinBaseTransaction) ContextCheck(para interfaces.Parameters) (map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.SetContextParameters(para); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	if err := a.HeightVersionCheck(); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	// check if duplicated with transaction in ledger
	if exist := a.IsTxHashDuplicate(*a.txHash); exist {
		log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	firstErr, end := a.SpecialContextCheck()
	if end {
		return nil, firstErr
	}

	return nil, nil
}

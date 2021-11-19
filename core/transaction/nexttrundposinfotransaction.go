// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type NextTurnDPOSInfoTransaction struct {
	BaseTransaction
}

func (t *NextTurnDPOSInfoTransaction) CheckTransactionInput() error {
	if len(t.sanityParameters.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *NextTurnDPOSInfoTransaction) CheckTransactionOutput() error {

	txn := t.sanityParameters.Transaction
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *NextTurnDPOSInfoTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("zero cost tx should have no attributes and programs")
	}
	return nil
}

func (t *NextTurnDPOSInfoTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *NextTurnDPOSInfoTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	nextTurnDPOSInfo, ok := t.Payload().(*payload.NextTurnDPOSInfo)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid NextTurnDPOSInfo payload")), true
	}

	if !blockchain.DefaultLedger.Arbitrators.IsNeedNextTurnDPOSInfo() {
		log.Warn("[checkNextTurnDPOSInfoTransaction] !IsNeedNextTurnDPOSInfo")
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should not have next turn dpos info transaction")), true
	}
	nextArbitrators := blockchain.DefaultLedger.Arbitrators.GetNextArbitrators()

	if !isNextArbitratorsSame(nextTurnDPOSInfo, nextArbitrators) {
		log.Warnf("[checkNextTurnDPOSInfoTransaction] CRPublicKeys %v, DPOSPublicKeys%v\n",
			convertToArbitersStr(nextTurnDPOSInfo.CRPublicKeys), convertToArbitersStr(nextTurnDPOSInfo.DPOSPublicKeys))

		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("checkNextTurnDPOSInfoTransaction nextTurnDPOSInfo was wrong")), true
	}
	return nil, true
}

func convertToArbitersStr(arbiters [][]byte) []string {
	var arbitersStr []string
	for _, v := range arbiters {
		arbitersStr = append(arbitersStr, common.BytesToHexString(v))
	}
	return arbitersStr
}

func isNextArbitratorsSame(nextTurnDPOSInfo *payload.NextTurnDPOSInfo,
	nextArbitrators []*state.ArbiterInfo) bool {
	if len(nextTurnDPOSInfo.CRPublicKeys)+len(nextTurnDPOSInfo.DPOSPublicKeys) != len(nextArbitrators) {
		log.Warn("[IsNextArbitratorsSame] nexArbitrators len ", len(nextArbitrators))
		return false
	}
	crindex := 0
	dposIndex := 0
	for _, v := range nextArbitrators {
		if blockchain.DefaultLedger.Arbitrators.IsNextCRCArbitrator(v.NodePublicKey) {
			if bytes.Equal(v.NodePublicKey, nextTurnDPOSInfo.CRPublicKeys[crindex]) ||
				(bytes.Equal([]byte{}, nextTurnDPOSInfo.CRPublicKeys[crindex]) &&
					!blockchain.DefaultLedger.Arbitrators.IsMemberElectedNextCRCArbitrator(v.NodePublicKey)) {
				crindex++
				continue
			} else {
				return false
			}
		} else {
			if bytes.Equal(v.NodePublicKey, nextTurnDPOSInfo.DPOSPublicKeys[dposIndex]) {
				dposIndex++
				continue
			} else {
				return false
			}
		}
	}
	return true
}

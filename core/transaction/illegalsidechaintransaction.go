// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type IllegalSideChainTransaction struct {
	BaseTransaction
}

func (t *IllegalSideChainTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.CheckTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.CheckTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
}

func (t *IllegalSideChainTransaction) CheckTransactionInput(params *TransactionParameters) error {
	if len(params.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *IllegalSideChainTransaction) CheckTransactionOutput(params *TransactionParameters) error {

	txn := params.Transaction
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *IllegalSideChainTransaction) CheckAttributeProgram(params *TransactionParameters) error {
	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("zero cost tx should have no attributes and programs")
	}
	return nil
}

func (t *IllegalSideChainTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	// todo add check after illegal side chain payload defained
	return errors.New("invalid payload type")
}

func (t *IllegalSideChainTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return true
}

func (t *IllegalSideChainTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	p, ok := t.Payload().(*payload.SidechainIllegalData)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if params.BlockChain.GetState().SpecialTxExists(t) {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("tx already exists")), true
	}

	if err := blockchain.CheckSidechainIllegalEvidence(p); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}

func CheckSidechainIllegalEvidence(p *payload.SidechainIllegalData) error {

	if p.IllegalType != payload.SidechainIllegalProposal &&
		p.IllegalType != payload.SidechainIllegalVote {
		return errors.New("invalid type")
	}

	_, err := crypto.DecodePoint(p.IllegalSigner)
	if err != nil {
		return err
	}

	if !blockchain.DefaultLedger.Arbitrators.IsArbitrator(p.IllegalSigner) {
		return errors.New("illegal signer is not one of current arbitrators")
	}

	_, err = common.Uint168FromAddress(p.GenesisBlockAddress)
	// todo check genesis block when sidechain registered in the future
	if err != nil {
		return err
	}

	if len(p.Signs) <= int(blockchain.DefaultLedger.Arbitrators.GetArbitersMajorityCount()) {
		return errors.New("insufficient signs count")
	}

	if p.Evidence.DataHash.Compare(p.CompareEvidence.DataHash) >= 0 {
		return errors.New("evidence order error")
	}

	//todo get arbitrators by payload.Height and verify each sign in signs

	return nil
}

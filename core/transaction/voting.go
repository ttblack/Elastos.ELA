// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

var MinVotesLockTime uint32 = 7200

type VotingTransaction struct {
	BaseTransaction
}

func (t *VotingTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *VotingTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Voting:
		return t.Payload().(*payload.Voting).Validate()

	}

	return errors.New("invalid payload type")
}

func (t *VotingTransaction) CheckAttributeProgram() error {
	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) != 1 {
		return errors.New("transaction should have only one program")
	}
	if t.Programs()[0].Code == nil {
		return fmt.Errorf("invalid program code nil")
	}
	if t.Programs()[0].Parameter == nil {
		return fmt.Errorf("invalid program parameter nil")
	}

	return nil
}

func (t *VotingTransaction) IsAllowedInPOWConsensus() bool {
	pld := t.Payload().(*payload.Voting)

	for _, vote := range pld.Contents {
		switch vote.VoteType {
		case outputpayload.Delegate:
		case outputpayload.CRC:
			log.Warn("not allow to vote CR in POW consensus")
			return false
		case outputpayload.CRCProposal:
			log.Warn("not allow to vote CRC proposal in POW consensus")
			return false
		case outputpayload.CRCImpeachment:
			log.Warn("not allow to vote CRImpeachment in POW consensus")
			return false
		}
	}

	inputProgramHashes := make(map[common.Uint168]struct{})
	for _, output := range t.references {
		inputProgramHashes[output.ProgramHash] = struct{}{}
	}
	outputProgramHashes := make(map[common.Uint168]struct{})
	for _, output := range t.Outputs() {
		outputProgramHashes[output.ProgramHash] = struct{}{}
	}
	for k, _ := range outputProgramHashes {
		if _, ok := inputProgramHashes[k]; !ok {
			log.Warn("output program hash is not in inputs")
			return false
		}
	}

	return true
}

func (t *VotingTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	// 1.check if the signer has vote rights and check if votes enough
	// 2.check different type of votes, enough? candidate exist?
	outputValue := t.Outputs()[0].Value
	blockHeight := t.parameters.BlockHeight
	crCommittee := t.parameters.BlockChain.GetCRCommittee()
	producers := t.parameters.BlockChain.GetState().GetActiveProducers()
	pds := getProducerPublicKeysMap(producers)
	pds2 := getDPoSV2ProducersMap(t.parameters.BlockChain.GetState().GetActivityV2Producers())

	// vote rights should be more than vote rights used in payload
	code := t.Programs()[0].Code
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	state := t.parameters.BlockChain.GetState()
	voteRights := state.DposV2VoteRights
	totalVotes, exist := voteRights[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no vote rights")), true
	}
	usedDPoSVoteRights, exist := state.DposVotes[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no DPoS vote rights")), true
	}
	usedDPoSV2VoteRights, exist := state.DposV2Votes[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no DPoS v2 vote rights")), true
	}
	usedCRVoteRights, exist := state.CRVotes[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no CR vote rights")), true
	}
	usedCRCProposalVoteRights, exist := state.CRCProposalVotes[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no CRCProposal vote rights")), true
	}
	usedCRImpeachmentVoteRights, exist := state.CRImpeachmentVotes[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no CRImpeachment vote rights")), true
	}

	var candidates []*crstate.Candidate
	if crCommittee.IsInVotingPeriod(blockHeight) {
		candidates = crCommittee.GetCandidates(crstate.Active)
	} else {
		candidates = []*crstate.Candidate{}
	}
	crs := getCRCIDsMap(candidates)

	pld := t.Payload().(*payload.Voting)
	switch t.PayloadVersion() {
	case payload.VoteVersion:
		for _, content := range pld.Contents {
			switch content.VoteType {
			case outputpayload.Delegate:
				err := t.checkVoteProducerContent(
					content, pds, outputValue, totalVotes-usedDPoSVoteRights)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.CRC:
				err := t.checkVoteCRContent(blockHeight,
					content, crs, outputValue, totalVotes-usedCRVoteRights)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.CRCProposal:
				err := t.checkVoteCRCProposalContent(
					content, outputValue, totalVotes-usedCRCProposalVoteRights)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.CRCImpeachment:
				err := t.checkCRImpeachmentContent(
					content, outputValue, totalVotes-usedCRImpeachmentVoteRights)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.DposV2:
				err := t.checkDPoSV2Content(content, pds2, outputValue, totalVotes-usedDPoSV2VoteRights)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			}
		}
	case payload.RenewalVoteVersion:
		for _, content := range pld.RenewalContents {
			producer := state.GetProducer(content.VotesInfo.Candidate)
			vote, err := producer.GetDetailDPoSV2Votes(*stakeProgramHash, content.ReferKey)
			if err != nil {
				return elaerr.Simple(elaerr.ErrTxPayload, err), true
			}
			if vote.VoteType != outputpayload.DposV2 {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid vote type")), true
			}
			if vote.BlockHeight < content.VotesInfo.LockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid lock time")), true
			}
			if vote.Info.Votes != content.VotesInfo.Votes {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("votes not equal")), true
			}
			if !bytes.Equal(vote.Info.Candidate, content.VotesInfo.Candidate) {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("candidate should be the same one")), true
			}
		}
	default:
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload version")), true
	}

	return nil, false
}

func (t *VotingTransaction) checkVoteProducerContent(content payload.VotesContent,
	pds map[string]struct{}, amount common.Fixed64, voteRights common.Fixed64) error {
	for _, cv := range content.VotesInfo {
		if _, ok := pds[common.BytesToHexString(cv.Candidate)]; !ok {
			return fmt.Errorf("invalid vote output payload "+
				"producer candidate: %s", common.BytesToHexString(cv.Candidate))
		}
	}
	var maxVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		if cv.Votes > amount {
			return errors.New("votes larger than output amount")
		}
		if maxVotes < cv.Votes {
			maxVotes = cv.Votes
		}
	}
	if maxVotes > voteRights {
		return errors.New("DPoS vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkVoteCRContent(blockHeight uint32,
	content payload.VotesContent, crs map[common.Uint168]struct{},
	amount common.Fixed64, voteRights common.Fixed64) error {

	if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(blockHeight) {
		return errors.New("cr vote tx must during voting period")
	}

	if blockHeight >= t.parameters.Config.CheckVoteCRCountHeight {
		if len(content.VotesInfo) > outputpayload.MaxVoteProducersPerTransaction {
			return errors.New("invalid count of CR candidates ")
		}
	}
	var totalVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		cid, err := common.Uint168FromBytes(cv.Candidate)
		if err != nil {
			return fmt.Errorf("invalid vote output payload " +
				"Candidate can not change to proper cid")
		}
		if _, ok := crs[*cid]; !ok {
			return fmt.Errorf("invalid vote output payload "+
				"CR candidate: %s", cid.String())
		}
		totalVotes += cv.Votes
	}
	if totalVotes > amount {
		return errors.New("total votes larger than output amount")
	}
	if totalVotes > voteRights {
		return errors.New("CR vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkVoteCRCProposalContent(
	content payload.VotesContent, amount common.Fixed64,
	voteRights common.Fixed64) error {
	var maxVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		if cv.Votes > amount {
			return errors.New("votes larger than output amount")
		}
		if maxVotes < cv.Votes {
			maxVotes = cv.Votes
		}
		proposalHash, err := common.Uint256FromBytes(cv.Candidate)
		if err != nil {
			return err
		}
		proposal := t.parameters.BlockChain.GetCRCommittee().GetProposal(*proposalHash)
		if proposal == nil || proposal.Status != crstate.CRAgreed {
			return fmt.Errorf("invalid CRCProposal: %s",
				common.ToReversedString(*proposalHash))
		}
	}

	if maxVotes > voteRights {
		return errors.New("CRCProposal vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkCRImpeachmentContent(content payload.VotesContent,
	amount common.Fixed64, voteRights common.Fixed64) error {
	crMembersMap := getCRMembersMap(t.parameters.BlockChain.GetCRCommittee().GetImpeachableMembers())
	var totalVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		if _, ok := crMembersMap[common.BytesToHexString(cv.Candidate)]; !ok {
			return errors.New("candidate should be one of the CR members")
		}
		totalVotes += cv.Votes
	}

	if totalVotes > amount {
		return errors.New("total votes larger than output amount")
	}
	if totalVotes > voteRights {
		return errors.New("CRImpeachment vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkDPoSV2Content(content payload.VotesContent,
	pds map[string]uint32, outputValue common.Fixed64, voteRights common.Fixed64) error {
	// totalVotes should be more than output value
	var totalVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		lockUntil, ok := pds[common.BytesToHexString(cv.Candidate)]
		if !ok {
			return fmt.Errorf("invalid vote output payload "+
				"producer candidate: %s", common.BytesToHexString(cv.Candidate))
		}
		if cv.LockTime > lockUntil || cv.LockTime-t.parameters.BlockHeight < MinVotesLockTime {
			return errors.New("invalid DPoS 2.0 votes lock time")
		}
		totalVotes += cv.Votes
	}
	if totalVotes > outputValue {
		return errors.New("votes larger than output amount")
	}
	if totalVotes > voteRights {
		return errors.New("DPoSV2 vote rights not enough")
	}

	return nil
}

// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RevertToPOWTransaction struct {
	BaseTransaction
}

func (t *RevertToPOWTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *RevertToPOWTransaction) HeightVersionCheck() error {
	if t.contextParameters.BlockHeight < t.contextParameters.Config.RevertToPOWStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before RevertToPOWStartHeight", t.TxType().Name()))
	}

	return nil
}

func (t *RevertToPOWTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	p, ok := t.Payload().(*payload.RevertToPOW)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if p.WorkingHeight != t.contextParameters.BlockHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid start POW block height")), true
	}

	switch p.Type {
	case payload.NoBlock:
		lastBlockTime := int64(t.contextParameters.BlockChain.BestChain.Timestamp)
		noBlockTime := t.contextParameters.Config.RevertToPOWNoBlockTime

		if t.contextParameters.TimeStamp == 0 {
			// is not in block, check by local time.
			localTime := t.MedianAdjustedTime().Unix()
			if localTime-lastBlockTime < noBlockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid block time")), true
			}
		} else {
			// is in block, check by the time of existed block.
			if int64(t.contextParameters.TimeStamp)-lastBlockTime < noBlockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid block time")), true
			}
		}
	case payload.NoProducers:
		if !t.contextParameters.BlockChain.GetState().NoProducers {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("current producers is enough")), true
		}
	case payload.NoClaimDPOSNode:
		if !t.contextParameters.BlockChain.GetState().NoClaimDPOSNode {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("current CR member claimed DPoS node")), true
		}
	}
	return nil, true
}

func (t *RevertToPOWTransaction) MedianAdjustedTime() time.Time {
	newTimestamp := t.contextParameters.BlockChain.TimeSource.AdjustedTime()
	minTimestamp := t.contextParameters.BlockChain.MedianTimePast.Add(time.Second)

	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}

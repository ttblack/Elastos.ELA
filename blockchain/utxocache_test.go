// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

var (
	utxoCacheDB *UtxoCacheDB
	utxoCache   *UTXOCache

	// refer tx hash: 160da301e49617c037ae9b630919af52b8ac458202cd64558af7e0dcc753e307
	referTx = &transactions.BaseTransaction{
		Version:        common2.TxVersion09,
		TxType:         common2.TransferAsset,
		PayloadVersion: 0,
		Payload:        &payload.TransferAsset{},
		Attributes:     []*common2.Attribute{},
		Inputs: []*common2.Input{
			{
				Previous: common2.OutPoint{
					Index: 0,
					TxID:  common.EmptyHash,
				},
				Sequence: 0,
			},
		},
		Outputs: []*common2.Output{
			{
				Value:   100,
				Type:    common2.OTVote,
				Payload: &outputpayload.VoteOutput{},
			},
		},
		LockTime: 5,
	}

	spendTx = &transactions.BaseTransaction{
		Inputs: []*common2.Input{
			{
				Previous: common2.OutPoint{
					Index: 0,
					TxID:  referTx.Hash(),
				},
				Sequence: 0,
			},
		},
	}
)

type UtxoCacheDB struct {
	transactions map[common.Uint256]*transactions.BaseTransaction
}

func init() {
	testing.Init()
}

func (s *UtxoCacheDB) GetTransaction(txID common.Uint256) (
	*transactions.BaseTransaction, uint32, error) {
	txn, exist := s.transactions[txID]
	if exist {
		return txn, 0, nil
	}
	return nil, 0, errors.New("leveldb: not found")
}

func (s *UtxoCacheDB) SetTransaction(txn *transactions.BaseTransaction) {
	s.transactions[txn.Hash()] = txn
}

func (s *UtxoCacheDB) RemoveTransaction(txID common.Uint256) {
	delete(s.transactions, txID)
}

func NewUtxoCacheDB() *UtxoCacheDB {
	var db UtxoCacheDB
	db.transactions = make(map[common.Uint256]*transactions.BaseTransaction)
	return &db
}

func TestUTXOCache_Init(t *testing.T) {
	utxoCacheDB = NewUtxoCacheDB()
	fmt.Println("refer tx hash:", referTx.Hash().String())
	utxoCacheDB.SetTransaction(referTx)
}

func TestUTXOCache_GetTxReferenceInfo(t *testing.T) {
	utxoCache = NewUTXOCache(utxoCacheDB, &config.DefaultParams)

	// get tx reference form db and cache it first time.
	reference, err := utxoCache.GetTxReference(spendTx)
	assert.NoError(t, err)
	for input, output := range reference {
		assert.Equal(t, referTx.Hash(), input.Previous.TxID)
		assert.Equal(t, uint16(0), input.Previous.Index)
		assert.Equal(t, uint32(0), input.Sequence)

		assert.Equal(t, common.Fixed64(100), output.Value)
		assert.Equal(t, common2.OTVote, output.Type)
	}

	// ensure above reference have been cached.
	utxoCacheDB.RemoveTransaction(referTx.Hash())
	_, _, err = utxoCacheDB.GetTransaction(referTx.Hash())
	assert.Equal(t, "leveldb: not found", err.Error())

	reference, err = utxoCache.GetTxReference(spendTx)
	assert.NoError(t, err)
	for input, output := range reference {
		assert.Equal(t, referTx.Hash(), input.Previous.TxID)
		assert.Equal(t, uint16(0), input.Previous.Index)
		assert.Equal(t, uint32(0), input.Sequence)

		assert.Equal(t, common.Fixed64(100), output.Value)
		assert.Equal(t, common2.OTVote, output.Type)
	}
}

func TestUTXOCache_CleanSpent(t *testing.T) {
	utxoCache.CleanTxCache()
	_, err := utxoCache.GetTransaction(spendTx.Hash())
	assert.Equal(t, "transaction not found, leveldb: not found", err.Error())
}

func TestUTXOCache_CleanCache(t *testing.T) {
	utxoCacheDB.SetTransaction(referTx)

	reference, err := utxoCache.GetTxReference(spendTx)
	assert.NoError(t, err)
	for input, output := range reference {
		assert.Equal(t, referTx.Hash(), input.Previous.TxID)
		assert.Equal(t, uint16(0), input.Previous.Index)
		assert.Equal(t, uint32(0), input.Sequence)

		assert.Equal(t, common.Fixed64(100), output.Value)
		assert.Equal(t, common2.OTVote, output.Type)
	}

	utxoCacheDB.RemoveTransaction(referTx.Hash())
	_, _, err = utxoCacheDB.GetTransaction(referTx.Hash())
	assert.Equal(t, "leveldb: not found", err.Error())

	utxoCache.CleanCache()
	_, err = utxoCache.GetTxReference(spendTx)
	assert.Equal(t,
		"GetTxReference failed, transaction not found, leveldb: not found",
		err.Error())
}

// Test for case that a map use pointer as a key
func Test_PointerKeyForMap(t *testing.T) {
	test.SkipShort(t)
	i1 := common2.Input{
		Previous: common2.OutPoint{
			TxID:  common.EmptyHash,
			Index: 15,
		},
		Sequence: 10,
	}

	i2 := common2.Input{
		Previous: common2.OutPoint{
			TxID:  common.EmptyHash,
			Index: 15,
		},
		Sequence: 10,
	}
	// ensure i1 and i2 have the same data
	assert.Equal(t, i1, i2)

	// pointer as a key
	m1 := make(map[*common2.Input]int)
	m1[&i1] = 1
	m1[&i2] = 2
	assert.Equal(t, 2, len(m1))
	//fmt.Println(m1)
	// NOTE: &i1 and &i2 are different keys in m1
	// map[{TxID: 0000000000000000000000000000000000000000000000000000000000000000 Index: 15 Sequence: 10}:1 {TxID: 0000000000000000000000000000000000000000000000000000000000000000 Index: 15 Sequence: 10}:2]

	// object as a key
	m2 := make(map[common2.Input]int)
	m2[i1] = 1
	m2[i2] = 2
	assert.Equal(t, 1, len(m2))
	//fmt.Println(m2)
	// map[{TxID: 0000000000000000000000000000000000000000000000000000000000000000 Index: 15 Sequence: 10}:2]

	// pointer as a key
	m4 := make(map[*int]int)
	i3 := 0
	i4 := 0
	m4[&i3] = 3
	m4[&i4] = 4
	assert.Equal(t, 2, len(m4))
	//fmt.Println(m4)
	// map[0xc0000b43d8:3 0xc0000b4400:4]
}

func TestUTXOCache_InsertReference(t *testing.T) {
	// init reference
	for i := uint32(0); i < uint32(maxReferenceSize); i++ {
		input := &common2.Input{
			Sequence: i,
		}
		output := &common2.Output{
			OutputLock: i,
		}
		utxoCache.insertReference(input, output)
	}
	assert.Equal(t, maxReferenceSize, len(utxoCache.reference))
	assert.Equal(t, maxReferenceSize, utxoCache.inputs.Len())
	assert.Equal(t, uint32(0), utxoCache.inputs.Front().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(maxReferenceSize-1), utxoCache.inputs.Back().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(0), utxoCache.reference[utxoCache.inputs.Front().Value.(common2.Input)].OutputLock)
	assert.Equal(t, uint32(maxReferenceSize-1), utxoCache.reference[utxoCache.inputs.Back().Value.(common2.Input)].OutputLock)

	for i := uint32(maxReferenceSize); i < uint32(maxReferenceSize+500); i++ {
		input := &common2.Input{
			Sequence: i,
		}
		output := &common2.Output{
			OutputLock: i,
		}
		utxoCache.insertReference(input, output)
	}
	assert.Equal(t, maxReferenceSize, len(utxoCache.reference))
	assert.Equal(t, maxReferenceSize, utxoCache.inputs.Len())
	assert.Equal(t, uint32(500), utxoCache.inputs.Front().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(maxReferenceSize+499), utxoCache.inputs.Back().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(500), utxoCache.reference[utxoCache.inputs.Front().Value.(common2.Input)].OutputLock)
	assert.Equal(t, uint32(maxReferenceSize+499), utxoCache.reference[utxoCache.inputs.Back().Value.(common2.Input)].
		OutputLock)
}

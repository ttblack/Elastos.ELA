// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"github.com/elastos/Elastos.ELA/common"
)

type CheckParameters struct {

	// for illegal proposal transaction
	SpecialTxExists func(txHash common.Uint256) bool
	GetCurrentArbitratorNodePublicKeys func(height uint32) [][]byte


	// transaction
	Transaction Transaction

	// others
	BlockHeight            uint32
	CRCommitteeStartHeight uint32
	ConsensusAlgorithm     byte
	DestroyELAAddress      common.Uint168
	CRAssetsAddress        common.Uint168
	FoundationAddress      common.Uint168
}

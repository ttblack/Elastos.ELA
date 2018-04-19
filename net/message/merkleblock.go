package message

import (
	"github.com/elastos/Elastos.ELA.Utility/bloom"
	"github.com/elastos/Elastos.ELA.Utility/core"
)

// NewMerkleBlock returns a new *MerkleBlock
func NewMerkleBlockMsg(block *core.Block, filter *bloom.Filter) ([]byte, error) {
	// Create and return the merkle block.
	merkleBlock := bloom.NewMerkleBlock(block, filter)

	body, err := merkleBlock.Serialize()
	if err != nil {
		return nil, err
	}

	return BuildMessage("merkleblock", body)
}
package prunable

import (
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/kvstore"
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp/packages/core/database"
)

const (
	blocksPrefix byte = iota
	rootBlocksPrefix
	attestationsPrefix
	ledgerStateDiffsPrefix
)

type Prunable struct {
	Blocks           *Blocks
	RootBlocks       *RootBlocks
	Attestations     func(index slot.Index) kvstore.KVStore
	LedgerStateDiffs func(index slot.Index) kvstore.KVStore
}

func New(dbManager *database.Manager) (newPrunable *Prunable) {
	return &Prunable{
		Blocks:           NewBlocks(dbManager, blocksPrefix),
		RootBlocks:       NewRootBlocks(dbManager, rootBlocksPrefix),
		Attestations:     lo.Bind([]byte{attestationsPrefix}, dbManager.Get),
		LedgerStateDiffs: lo.Bind([]byte{ledgerStateDiffsPrefix}, dbManager.Get),
	}
}

package proofs

/*
import (
	"github.com/celestiaorg/smt"
	"github.com/pkg/errors"
	"github.com/izuc/zipp.foundation/lo"
	"golang.org/x/crypto/blake2b"
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp/packages/protocol/ledger"
	"github.com/izuc/zipp/packages/protocol/ledger/utxo"
	"github.com/izuc/zipp/packages/protocol/models"
)

// region proofs helpers ///////////////////////////////////////////////////////////////////////////////////////////////

// CommitmentProof represents an inclusion proof for a specific slot.
type CommitmentProof struct {
	SlotIndex    slot.Index
	proof smt.SparseMerkleProof
	root  []byte
}

// GetBlockInclusionProof gets the proof of the inclusion (acceptance) of a block.
func (m *Manager) GetBlockInclusionProof(blockID models.BlockID) (*CommitmentProof, error) {
	var ei slot.Index
	block, exists := m.mesh.BlockDAG.Block(blockID)
	if !exists {
		return nil, errors.Errorf("cannot retrieve block with id %s", blockID)
	}
	t := block.IssuingTime()
	ei = slot.IndexFromTime(t)
	proof, err := m.commitmentFactory.ProofMeshRoot(ei, blockID)
	if err != nil {
		return nil, err
	}
	return proof, nil
}

// GetTransactionInclusionProof gets the proof of the inclusion (acceptance) of a transaction.
func (m *Manager) GetTransactionInclusionProof(transactionID utxo.TransactionID) (*CommitmentProof, error) {
	var ei slot.Index
	m.ledger.Storage.CachedTransactionMetadata(transactionID).Consume(func(txMeta *ledger.TransactionMetadata) {
		ei = slot.IndexFromTime(txMeta.InclusionTime())
	})
	proof, err := m.commitmentFactory.ProofStateMutationRoot(ei, transactionID)
	if err != nil {
		return nil, err
	}
	return proof, nil
}

func (f *commitmentFactory) verifyRoot(proof CommitmentProof, key []byte, value []byte) bool {
	return smt.VerifyProof(proof.proof, proof.root, key, value, lo.PanicOnErr(blake2b.New256(nil)))
}

// ProofStateRoot returns the merkle proof for the outputID against the state root.
func (f *commitmentFactory) ProofStateRoot(ei slot.Index, outID utxo.OutputID) (*CommitmentProof, error) {
	key := outID.Bytes()
	root, exists := f.commitmentTrees.Get(ei)
	if !exists {
		return nil, errors.Errorf("could not obtain commitment trees for slot %d", ei)
	}
	meshRoot := root.meshTree.Root()
	proof, err := f.stateRootTree.ProveForRoot(key, meshRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate the state root proof")
	}
	return &CommitmentProof{ei, proof, meshRoot}, nil
}

// ProofStateMutationRoot returns the merkle proof for the transactionID against the state mutation root.
func (f *commitmentFactory) ProofStateMutationRoot(ei slot.Index, txID utxo.TransactionID) (*CommitmentProof, error) {
	committmentTrees, err := f.getCommitmentTrees(ei)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get commitment trees for slot %d", ei)
	}

	key := txID.Bytes()
	root := committmentTrees.stateMutationTree.Root()
	proof, err := committmentTrees.stateMutationTree.ProveForRoot(key, root)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate the state mutation root proof")
	}
	return &CommitmentProof{ei, proof, root}, nil
}

// ProofMeshRoot returns the merkle proof for the blockID against the mesh root.
func (f *commitmentFactory) ProofMeshRoot(ei slot.Index, blockID models.BlockID) (*CommitmentProof, error) {
	committmentTrees, err := f.getCommitmentTrees(ei)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get commitment trees for slot %d", ei)
	}

	key, _ := blockID.Bytes()
	root := committmentTrees.meshTree.Root()
	proof, err := committmentTrees.meshTree.ProveForRoot(key, root)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate the mesh root proof")
	}
	return &CommitmentProof{ei, proof, root}, nil
}

// VerifyMeshRoot verify the provided merkle proof against the mesh root.
func (f *commitmentFactory) VerifyMeshRoot(proof CommitmentProof, blockID models.BlockID) bool {
	key, _ := blockID.Bytes()
	return f.verifyRoot(proof, key, key)
}

// VerifyStateMutationRoot verify the provided merkle proof against the state mutation root.
func (f *commitmentFactory) VerifyStateMutationRoot(proof CommitmentProof, transactionID utxo.TransactionID) bool {
	key := transactionID.Bytes()
	return f.verifyRoot(proof, key, key)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

*/

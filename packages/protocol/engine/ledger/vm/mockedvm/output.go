package mockedvm

import (
	"github.com/izuc/zipp.foundation/objectstorage/generic/model"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/utxo"
)

// MockedOutput is the container for the data produced by executing a MockedTransaction.
type MockedOutput struct {
	model.Storable[utxo.OutputID, MockedOutput, *MockedOutput, mockedOutput] `serix:"0"`
}

type mockedOutput struct {
	// TxID contains the identifier of the Transaction that created this MockedOutput.
	TxID utxo.TransactionID `serix:"0"`

	// Index contains the Index of the Output in respect to it's creating Transaction (the nth Output will have the
	// Index n).
	Index uint16 `serix:"1"`

	Balance uint64 `serix:"2"`
}

// NewMockedOutput creates a new MockedOutput based on the utxo.TransactionID and its index within the MockedTransaction.
func NewMockedOutput(txID utxo.TransactionID, index uint16, balance uint64) (out *MockedOutput) {
	out = model.NewStorable[utxo.OutputID, MockedOutput](&mockedOutput{
		TxID:    txID,
		Index:   index,
		Balance: balance,
	})
	out.SetID(utxo.OutputID{TransactionID: txID, Index: index})
	return out
}

// code contract (make sure the struct implements all required methods).
var _ utxo.Output = new(MockedOutput)

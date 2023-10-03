package wallet

import (
	"github.com/izuc/zipp.foundation/core/types/confirmation"

	"github.com/izuc/zipp/client/wallet/packages/address"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/mana"
)

// Connector represents an interface that defines how the wallet interacts with the network. A wallet can either be used
// locally on a server or it can connect remotely using the web API.
type Connector interface {
	UnspentOutputs(addresses ...address.Address) (unspentOutputs OutputsByAddressAndOutputID, err error)
	SendTransaction(transaction *devnetvm.Transaction) (err error)
	RequestFaucetFunds(address address.Address, powTarget int) (err error)
	GetAllowedPledgeIDs() (pledgeIDMap map[mana.Type][]string, err error)
	GetTransactionConfirmationState(txID utxo.TransactionID) (confirmationState confirmation.State, err error)
	GetUnspentAliasOutput(address *devnetvm.AliasAddress) (output *devnetvm.AliasOutput, err error)
}

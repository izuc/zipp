package seed

import (
	"github.com/izuc/zipp.foundation/crypto/ed25519"
	"github.com/izuc/zipp/client/wallet/packages/address"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/vm/devnetvm"
)

// Seed represents a seed for ZIPP wallets. A seed allows us to generate a deterministic sequence of Addresses and their
// corresponding KeyPairs.
type Seed struct {
	*ed25519.Seed
}

// NewSeed is the factory method for an ZIPP seed. It either generates a new one or imports an existing  marshaled seed.
// before.
func NewSeed(optionalSeedBytes ...[]byte) *Seed {
	return &Seed{
		ed25519.NewSeed(optionalSeedBytes...),
	}
}

// Address returns an Address which can be used for receiving or sending funds.
func (seed *Seed) Address(index uint64) (addr address.Address) {
	addr = address.Address{
		Index: index,
	}
	copy(addr.AddressBytes[:], devnetvm.NewED25519Address(seed.Seed.KeyPair(index).PublicKey).Bytes())

	return
}

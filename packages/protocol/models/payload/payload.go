package payload

import (
	"context"

	"github.com/iotaledger/hive.go/core/serix"
	"github.com/pkg/errors"
)

// MaxSize = MaxBlockSize -
//
//	                   (version(1) + parentsBlocksCount(1) + 3 * (parentsType(1) + parentsCount(1) + 8 * reference(40)) +
//			      issuerPK(32) + issuanceTime(8) + seqNum(8) + payloadLength(4) +
//			  + ECRecordI(8) + RootsID(32) + PrevID(32) + LatestConfirmedEpoch(8)
//			  + nonce(8) + signature(64)
//			      = MaxBlockSize - 1172 bytes = 64364
const MaxSize = 65536 - 1172

// Payload represents the generic interface for an object that can be embedded in Blocks of the Tangle.
type Payload interface {
	// Type returns the Type of the Payload.
	Type() Type

	// Bytes returns a marshaled version of the Payload.
	Bytes() ([]byte, error)

	// String returns a human-readable version of the Payload.
	String() string
}

// FromBytes unmarshals a Payload from a sequence of bytes.
func PayloadTypeFromBytes(data []byte) (payloadType Type, consumedBytes int, err error) {
	_, err = serix.DefaultAPI.Decode(context.Background(), data, &payloadType)
	if err != nil {
		err = errors.Errorf("failed to parse PayloadType (%v)", err)
		return
	}

	return
}

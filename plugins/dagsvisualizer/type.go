package dagsvisualizer

import (
	"github.com/izuc/zipp/packages/app/jsonmodels"
)

const (
	// BlkTypeMeshVertex is the type of the Mesh DAG vertex.
	BlkTypeMeshVertex byte = iota
	// BlkTypeMeshBooked is the type of the Mesh DAG confirmed block.
	BlkTypeMeshBooked
	// BlkTypeMeshConfirmed is the type of the Mesh DAG confirmed block.
	BlkTypeMeshConfirmed
	// BlkTypeMeshTxConfirmationState is the type of the Mesh DAG transaction ConfirmationState.
	BlkTypeMeshTxConfirmationState
	// BlkTypeUTXOVertex is the type of the UTXO DAG vertex.
	BlkTypeUTXOVertex
	// BlkTypeUTXOBooked is the type of the booked transaction.
	BlkTypeUTXOBooked
	// BlkTypeUTXOConfirmationStateChanged is the type of the UTXO DAG vertex confirmation state block.
	BlkTypeUTXOConfirmationStateChanged
	// BlkTypeConflictVertex is the type of the conflict DAG vertex.
	BlkTypeConflictVertex
	// BlkTypeConflictParentsUpdate is the type of the conflict DAG vertex parents updated block.
	BlkTypeConflictParentsUpdate
	// BlkTypeConflictConfirmationStateChanged is the type of the conflict DAG vertex confirmed block.
	BlkTypeConflictConfirmationStateChanged
	// BlkTypeConflictWeightChanged is the type of the conflict DAG vertex weight changed block.
	BlkTypeConflictWeightChanged
)

type wsBlock struct {
	Type byte        `json:"type"`
	Data interface{} `json:"data"`
}

type meshVertex struct {
	ID                    string   `json:"ID"`
	StrongParentIDs       []string `json:"strongParentIDs"`
	WeakParentIDs         []string `json:"weakParentIDs"`
	ShallowLikeParentIDs  []string `json:"shallowLikeParentIDs"`
	ConflictIDs           []string `json:"conflictIDs"`
	IsMarker              bool     `json:"isMarker"`
	IsTx                  bool     `json:"isTx"`
	TxID                  string   `json:"txID,omitempty"`
	IsConfirmed           bool     `json:"isConfirmed"`
	ConfirmationStateTime int64    `json:"confirmationStateTime"`
	ConfirmationState     string   `json:"confirmationState,omitempty"`
}

type meshBooked struct {
	ID          string   `json:"ID"`
	IsMarker    bool     `json:"isMarker"`
	ConflictIDs []string `json:"conflictIDs"`
}

type meshConfirmed struct {
	ID           string `json:"ID"`
	Accepted     bool   `json:"confirmationState"`
	AcceptedTime int64  `json:"confirmationStateTime"`
}

type meshTxConfirmationStateChanged struct {
	ID          string `json:"ID"`
	IsConfirmed bool   `json:"isConfirmed"`
}

type utxoVertex struct {
	BlkID                 string              `json:"blkID"`
	ID                    string              `json:"ID"`
	Inputs                []*jsonmodels.Input `json:"inputs"`
	Outputs               []string            `json:"outputs"`
	IsConfirmed           bool                `json:"isConfirmed"`
	ConfirmationState     string              `json:"confirmationState"`
	ConflictIDs           []string            `json:"conflictIDs"`
	ConfirmationStateTime int64               `json:"confirmationStateTime"`
}

type utxoBooked struct {
	ID          string   `json:"ID"`
	ConflictIDs []string `json:"conflictIDs"`
}

type utxoConfirmationStateChanged struct {
	ID                    string `json:"ID"`
	ConfirmationState     string `json:"confirmationState"`
	ConfirmationStateTime int64  `json:"confirmationStateTime"`
	IsConfirmed           bool   `json:"isConfirmed"`
}

type conflictVertex struct {
	ID                string                                   `json:"ID"`
	Parents           []string                                 `json:"parents"`
	IsConfirmed       bool                                     `json:"isConfirmed"`
	Conflicts         *jsonmodels.GetConflictConflictsResponse `json:"conflicts"`
	ConfirmationState string                                   `json:"confirmationState"`
	AW                int64                                    `json:"aw"`
}

type conflictParentUpdate struct {
	ID      string   `json:"ID"`
	Parents []string `json:"parents"`
}

type conflictConfirmationStateChanged struct {
	ID                string `json:"ID"`
	ConfirmationState string `json:"confirmationState"`
	IsConfirmed       bool   `json:"isConfirmed"`
}

type conflictWeightChanged struct {
	ID                string `json:"ID"`
	Weight            int64  `json:"weight"`
	ConfirmationState string `json:"confirmationState"`
}

type searchResult struct {
	Blocks    []*meshVertex     `json:"blocks"`
	Txs       []*utxoVertex     `json:"txs"`
	Conflicts []*conflictVertex `json:"conflicts"`
	Error     string            `json:"error,omitempty"`
}

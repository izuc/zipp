package mana

import (
	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/mr-tron/base58"
)

// Node represents a node and its mana value.
type Node struct {
	ID   identity.ID
	Mana float64
}

// NodeStr defines a node and its mana value.
// The node ID is stringified.
type NodeStr struct {
	ShortNodeID string  `json:"shortNodeID"`
	NodeID      string  `json:"nodeID"`
	Mana        float64 `json:"mana"`
}

// ToNodeStr converts a Node to a NodeStr
func (n Node) ToNodeStr() NodeStr {
	bytes, err := n.ID.Bytes()
	if err != nil {
		// Handle the error appropriately.
		// Depending on your design, you may want to return an error or use a default value.
		panic(err) // this is just an example, handle it as suits your application
	}
	return NodeStr{
		ShortNodeID: n.ID.String(),
		NodeID:      base58.Encode(bytes),
		Mana:        n.Mana,
	}
}

// NodeMap is a map of nodeID and mana value.
type NodeMap map[identity.ID]float64

// NodeMapStr is a NodeMap but with string id.
type NodeMapStr map[string]float64

// ToNodeStrList converts a NodeMap to a list of NodeStr.
func (n NodeMap) ToNodeStrList() []NodeStr {
	var list []NodeStr
	for ID, val := range n {
		bytes, err := ID.Bytes()
		if err != nil {
			// Handle the error appropriately.
			// Depending on your design, you may want to return an error or use a default value.
			panic(err) // this is just an example, handle it as suits your application
		}
		list = append(list, NodeStr{
			ShortNodeID: ID.String(),
			NodeID:      base58.Encode(bytes),
			Mana:        val,
		})
	}
	return list
}

// GetPercentile returns the top percentile the node belongs to relative to the network in terms of mana.
func (n NodeMap) GetPercentile(node identity.ID) (float64, error) {
	if len(n) == 0 {
		return 0, nil
	}
	value, ok := n[node]
	if !ok {
		return 0, ErrNodeNotFoundInBaseManaVector
	}
	nBelow := 0.0
	for _, val := range n {
		if val < value {
			nBelow++
		}
	}

	return (nBelow / float64(len(n))) * 100, nil
}

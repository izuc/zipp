package client

import (
	"io"
	"strings"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/mr-tron/base58"

	"github.com/izuc/zipp/packages/app/metrics"
	"github.com/izuc/zipp/plugins/analysis/packet"
	"github.com/izuc/zipp/plugins/banner"
)

// EventDispatchers holds the Heartbeat function.
type EventDispatchers struct {
	// Heartbeat defines the Heartbeat function.
	Heartbeat func(heartbeat *packet.Heartbeat)
}

func sendHeartbeat(w io.Writer, hb *packet.Heartbeat) {
	var out strings.Builder
	for _, value := range hb.OutboundIDs {
		out.WriteString(base58.Encode(value))
	}
	var in strings.Builder
	for _, value := range hb.InboundIDs {
		in.WriteString(base58.Encode(value))
	}
	log.Debugw(
		"Heartbeat",
		"networkID", string(hb.NetworkID),
		"nodeID", base58.Encode(hb.OwnID),
		"outboundIDs", out.String(),
		"inboundIDs", in.String(),
	)

	data, err := packet.NewHeartbeatBlock(hb)
	if err != nil {
		log.Info(err, " - heartbeat block skipped")
		return
	}

	if _, err = w.Write(data); err != nil {
		log.Debugw("Error while writing to connection", "Description", err)
	}
	// trigger AnalysisOutboundBytes event
	metrics.Events.AnalysisOutboundBytes.Trigger(&metrics.AnalysisOutboundBytesEvent{AmountBytes: uint64(len(data))})
}

func createHeartbeat() *packet.Heartbeat {
	// get own ID
	nodeID := make([]byte, len(identity.ID{}))
	if deps.Local != nil {
		nodeIDBytes, err := deps.Local.ID().Bytes()
		if err != nil {
			log.Error("Failed to get bytes from node ID: ", err)
			return nil // or handle the error as appropriate
		}
		copy(nodeID, nodeIDBytes)
	}

	var outboundIDs [][]byte
	var inboundIDs [][]byte

	// get outboundIDs (chosen neighbors)
	outgoingNeighbors := deps.Selection.GetOutgoingNeighbors()
	outboundIDs = make([][]byte, len(outgoingNeighbors))
	for i, neighbor := range outgoingNeighbors {
		outboundIDs[i] = make([]byte, len(identity.ID{}))
		neighborIDBytes, err := neighbor.ID().Bytes()
		if err != nil {
			log.Error("Failed to get bytes from neighbor ID: ", err)
			continue // or handle the error as appropriate
		}
		copy(outboundIDs[i], neighborIDBytes) // for the outgoingNeighbors loop
	}

	// get inboundIDs (accepted neighbors)
	incomingNeighbors := deps.Selection.GetIncomingNeighbors()
	inboundIDs = make([][]byte, len(incomingNeighbors))
	for i, neighbor := range incomingNeighbors {
		inboundIDs[i] = make([]byte, len(identity.ID{}))
		neighborIDBytes, err := neighbor.ID().Bytes()
		if err != nil {
			log.Error("Failed to get bytes from neighbor ID: ", err)
			continue // or handle the error as appropriate
		}
		copy(outboundIDs[i], neighborIDBytes) // for the outgoingNeighbors loop
	}

	return &packet.Heartbeat{NetworkID: []byte(banner.SimplifiedAppVersion), OwnID: nodeID, OutboundIDs: outboundIDs, InboundIDs: inboundIDs}
}

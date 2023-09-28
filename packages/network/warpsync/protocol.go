package warpsync

import (
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/logger"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/network"
	wp "github.com/izuc/zipp/packages/network/warpsync/proto"
)

const (
	protocolID = "warpsync/0.0.1"
)

type Protocol struct {
	Events *Events

	workerPool      *workerpool.WorkerPool
	networkEndpoint network.Endpoint
	log             *logger.Logger
}

func New(workerPool *workerpool.WorkerPool, networkEndpoint network.Endpoint, log *logger.Logger) (protocol *Protocol) {
	protocol = &Protocol{
		Events:          NewEvents(),
		workerPool:      workerPool,
		networkEndpoint: networkEndpoint,
		log:             log,
	}

	protocol.networkEndpoint.RegisterProtocol(protocolID, warpSyncPacketFactory, protocol.handlePacket)

	return
}

func (p *Protocol) Stop() {
	p.networkEndpoint.UnregisterProtocol(protocolID)
}

func (p *Protocol) handlePacket(id identity.ID, packet proto.Message) error {
	wpPacket := packet.(*wp.Packet)
	switch packetBody := wpPacket.GetBody().(type) {
	case *wp.Packet_SlotBlocksRequest:
		submitTask(p.workerPool, p.processSlotBlocksRequestPacket, packetBody, id)
	case *wp.Packet_SlotBlocksStart:
		submitTask(p.workerPool, p.processSlotBlocksStartPacket, packetBody, id)
	case *wp.Packet_SlotBlocksBatch:
		submitTask(p.workerPool, p.processSlotBlocksBatchPacket, packetBody, id)
	case *wp.Packet_SlotBlocksEnd:
		submitTask(p.workerPool, p.processSlotBlocksEndPacket, packetBody, id)
	default:
		return errors.Errorf("unsupported packet; packet=%+v, packetBody=%T-%+v", wpPacket, packetBody, packetBody)
	}

	return nil
}

func warpSyncPacketFactory() proto.Message {
	return &wp.Packet{}
}

func submitTask[P any](wp *workerpool.WorkerPool, packetProcessor func(packet P, id identity.ID), packet P, id identity.ID) {
	wp.Submit(func() { packetProcessor(packet, id) })
}

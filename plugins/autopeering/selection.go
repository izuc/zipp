package autopeering

import (
	"github.com/izuc/zipp.foundation/autopeering/discover"
	"github.com/izuc/zipp.foundation/autopeering/peer"
	"github.com/izuc/zipp.foundation/autopeering/peer/service"
	"github.com/izuc/zipp.foundation/autopeering/selection"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/logger"
)

func createPeerSel(localID *peer.Local, nbrDiscover *discover.Protocol) *selection.Protocol {
	// assure that the logger is available
	log := logger.NewLogger(PluginName).Named("sel")

	log.Infof("createPeerSel function called with localID: %s", localID.ID())

	return selection.New(localID, nbrDiscover,
		selection.Logger(log),
		selection.NeighborValidator(selection.ValidatorFunc(isValidNeighbor)),
		selection.UseMana(Parameters.Mana),
		selection.ManaFunc(evalMana),
		selection.R(Parameters.R),
		selection.Ro(Parameters.Ro),
	)
}

// isValidNeighbor checks whether a peer is a valid neighbor.
func isValidNeighbor(p *peer.Peer) bool {
	// gossip must be supported
	gossipService := p.Services().Get(service.P2PKey)
	if gossipService == nil {
		logger.NewLogger(PluginName).Debugf("Peer %s rejected: gossip service not supported.", p.ID())
		return false
	}
	// gossip service must be valid
	if gossipService.Network() != "tcp4" || gossipService.Port() < 0 || gossipService.Port() > 65535 {
		logger.NewLogger(PluginName).Debugf("Peer %s rejected: invalid gossip service.", p.ID())
		return false
	}
	return true
}

func evalMana(nodeIdentity *identity.Identity) uint64 {
	if deps.ManaFunc == nil {
		return 0
	}
	m, _, err := deps.ManaFunc(nodeIdentity.ID())
	if err != nil {
		return 0
	}
	return uint64(m)
}

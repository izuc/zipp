package mana

import (
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/izuc/zipp.foundation/autopeering/peer"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/ds/advancedset"
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp/packages/app/jsonmodels"
)

func getOnlineAccessHandler(c echo.Context) error {
	resp := make([]*jsonmodels.OnlineIssuerStr, 0)
	manaMap := deps.Protocol.Engine().ThroughputQuota.BalanceByIDs()
	var knownPeers *advancedset.AdvancedSet[identity.ID]
	if deps.Discovery != nil {
		knownPeers = advancedset.New[identity.ID](lo.Map(deps.Discovery.GetVerifiedPeers(), func(p *peer.Peer) identity.ID {
			return p.ID()
		})...)
	}

	for p, manaValue := range manaMap {
		if knownPeers != nil && !knownPeers.Has(p) && p != deps.Local.ID() {
			continue
		}

		resp = append(resp, &jsonmodels.OnlineIssuerStr{
			ShortID: p.String(),
			ID:      p.EncodeBase58(),
			Mana:    manaValue,
		})
	}

	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Mana > resp[j].Mana || (resp[i].Mana == resp[j].Mana && resp[i].ID > resp[j].ID)
	})
	for rank, onlineIssuer := range resp {
		onlineIssuer.OnlineRank = rank + 1
	}

	return c.JSON(http.StatusOK, jsonmodels.GetOnlineResponse{
		Online:    resp,
		Timestamp: time.Now().Unix(),
	})
}

func getOnlineConsensusHandler(c echo.Context) error {
	resp := make([]*jsonmodels.OnlineIssuerStr, 0)
	manaMap := lo.PanicOnErr(deps.Protocol.Engine().SybilProtection.Validators().Weights.Map())
	for p, manaValue := range manaMap {
		resp = append(resp, &jsonmodels.OnlineIssuerStr{
			ShortID: p.String(),
			ID:      p.EncodeBase58(),
			Mana:    manaValue,
		})
	}

	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Mana > resp[j].Mana || resp[i].ID > resp[j].ID
	})
	for rank, onlineIssuer := range resp {
		onlineIssuer.OnlineRank = rank + 1
	}

	return c.JSON(http.StatusOK, jsonmodels.GetOnlineResponse{
		Online:    resp,
		Timestamp: time.Now().Unix(),
	})
}

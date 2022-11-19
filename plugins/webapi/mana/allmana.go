package mana

import (
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo"

	"github.com/iotaledger/goshimmer/packages/app/jsonmodels"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/manatracker/manamodels"
)

// getAllManaHandler handles the request.
func getAllManaHandler(c echo.Context) error {
	access := deps.Protocol.Engine().ManaTracker.ManaByIDs()
	accessList := manamodels.IssuerMap(access).ToIssuerStrList()
	sort.Slice(accessList, func(i, j int) bool {
		return accessList[i].Mana > accessList[j].Mana
	})
	consensusList := manamodels.IssuerMap(deps.Protocol.Engine().SybilProtection.Weights().AsMap()).ToIssuerStrList()
	sort.Slice(consensusList, func(i, j int) bool {
		return consensusList[i].Mana > consensusList[j].Mana
	})
	return c.JSON(http.StatusOK, jsonmodels.GetAllManaResponse{
		Access:             accessList,
		AccessTimestamp:    time.Now().Unix(),
		Consensus:          consensusList,
		ConsensusTimestamp: time.Now().Unix(),
	})
}

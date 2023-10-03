package mana

import (
	"net/http"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/mr-tron/base58"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mana"
	manaPlugin "github.com/izuc/zipp/plugins/blocklayer"
)

// Handler handles the request.
func allowedManaPledgeHandler(c echo.Context) error {
	access := manaPlugin.GetAllowedPledgeNodes(mana.AccessMana)
	var accessNodes []string
	access.Allowed.ForEach(func(element identity.ID) {
		bytes, err := element.Bytes()
		if err != nil {
			// Handle error. For now, we'll simply log and skip.
			log.Warnf("Failed to get bytes for identity: %s", err)
			return
		}
		accessNodes = append(accessNodes, base58.Encode(bytes))
	})
	if len(accessNodes) == 0 {
		return c.JSON(http.StatusNotFound, jsonmodels.AllowedManaPledgeResponse{Error: "No access mana pledge IDs are accepted"})
	}

	consensus := manaPlugin.GetAllowedPledgeNodes(mana.ConsensusMana)
	var consensusNodes []string
	consensus.Allowed.ForEach(func(element identity.ID) {
		bytes, err := element.Bytes()
		if err != nil {
			// Handle error. For now, we'll simply log and skip.
			log.Warnf("Failed to get bytes for identity: %s", err)
			return
		}
		consensusNodes = append(consensusNodes, base58.Encode(bytes))
	})
	if len(consensusNodes) == 0 {
		return c.JSON(http.StatusNotFound, jsonmodels.AllowedManaPledgeResponse{Error: "No consensus mana pledge IDs are accepted"})
	}

	return c.JSON(http.StatusOK, jsonmodels.AllowedManaPledgeResponse{
		Access: jsonmodels.AllowedPledge{
			IsFilterEnabled: access.IsFilterEnabled,
			Allowed:         accessNodes,
		},
		Consensus: jsonmodels.AllowedPledge{
			IsFilterEnabled: consensus.IsFilterEnabled,
			Allowed:         consensusNodes,
		},
	})
}

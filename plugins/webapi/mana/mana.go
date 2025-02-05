package mana

import (
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mana"
	manaPlugin "github.com/izuc/zipp/plugins/blocklayer"
)

// getManaHandler handles the request.
func getManaHandler(c echo.Context) error {
	var request jsonmodels.GetManaRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.GetManaResponse{Error: err.Error()})
	}
	ID, err := mana.IDFromStr(request.NodeID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.GetManaResponse{Error: err.Error()})
	}
	if request.NodeID == "" {
		ID = deps.Local.ID()
	}
	t := time.Now()
	accessMana, tAccess, err := manaPlugin.GetAccessMana(ID, t)
	if err != nil {
		if errors.Is(err, mana.ErrNodeNotFoundInBaseManaVector) {
			accessMana = 0
			tAccess = t
		} else {
			return c.JSON(http.StatusBadRequest, jsonmodels.GetManaResponse{Error: err.Error()})
		}
	}
	consensusMana, tConsensus, err := manaPlugin.GetConsensusMana(ID, t)
	if err != nil {
		if errors.Is(err, mana.ErrNodeNotFoundInBaseManaVector) {
			consensusMana = 0
			tConsensus = t
		} else {
			return c.JSON(http.StatusBadRequest, jsonmodels.GetManaResponse{Error: err.Error()})
		}
	}

	bytes, err := ID.Bytes()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, jsonmodels.ErrorResponse{Error: "Failed to get bytes for identity"})
	}

	return c.JSON(http.StatusOK, jsonmodels.GetManaResponse{
		ShortNodeID:        ID.String(),
		NodeID:             base58.Encode(bytes),
		Access:             accessMana,
		AccessTimestamp:    tAccess.Unix(),
		Consensus:          consensusMana,
		ConsensusTimestamp: tConsensus.Unix(),
	})
}

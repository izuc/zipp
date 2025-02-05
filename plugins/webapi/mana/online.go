package mana

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/mr-tron/base58"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mana"
	manaPlugin "github.com/izuc/zipp/plugins/blocklayer"
)

func getOnlineAccessHandler(c echo.Context) error {
	return getOnlineHandler(c, mana.AccessMana)
}

func getOnlineConsensusHandler(c echo.Context) error {
	return getOnlineHandler(c, mana.ConsensusMana)
}

// getOnlineHandler handles the request.
func getOnlineHandler(c echo.Context, manaType mana.Type) error {
	onlinePeersMana, t, err := manaPlugin.GetOnlineNodes(manaType)
	if err != nil {
		return c.JSON(http.StatusNotFound, jsonmodels.GetOnlineResponse{Error: err.Error()})
	}

	resp := make([]jsonmodels.OnlineNodeStr, 0)
	for index, value := range onlinePeersMana {
		bytes, err := value.ID.Bytes()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, jsonmodels.ErrorResponse{Error: "Failed to get bytes for identity"})
		}
		resp = append(resp, jsonmodels.OnlineNodeStr{
			OnlineRank: index + 1,
			ShortID:    value.ID.String(),
			ID:         base58.Encode(bytes),
			Mana:       value.Mana,
		})
	}

	return c.JSON(http.StatusOK, jsonmodels.GetOnlineResponse{
		Online:    resp,
		Timestamp: t.Unix(),
	})
}

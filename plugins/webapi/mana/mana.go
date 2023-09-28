package mana

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mr-tron/base58"

	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp/packages/app/jsonmodels"
)

// getManaHandler handles the request.
func getManaHandler(c echo.Context) error {
	var request jsonmodels.GetManaRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.GetManaResponse{Error: err.Error()})
	}

	IDstr := request.IssuerID
	if IDstr == "" {
		IDstr = deps.Local.ID().EncodeBase58()
	}

	ID, err := identity.DecodeIDBase58(IDstr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.GetManaResponse{Error: err.Error()})
	}

	accessMana, _ := deps.Protocol.Engine().ThroughputQuota.Balance(ID)
	consensusMana := lo.Return1(deps.Protocol.Engine().SybilProtection.Weights().Get(ID)).Value

	return c.JSON(http.StatusOK, jsonmodels.GetManaResponse{
		ShortIssuerID:      ID.String(),
		IssuerID:           base58.Encode(lo.PanicOnErr(ID.Bytes())),
		Access:             accessMana,
		AccessTimestamp:    time.Now().Unix(),
		Consensus:          consensusMana,
		ConsensusTimestamp: time.Now().Unix(),
	})
}

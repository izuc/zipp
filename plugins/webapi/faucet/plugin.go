package faucet

import (
	"net/http"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"go.uber.org/dig"

	faucetpkg "github.com/izuc/zipp/packages/app/faucet"
	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/mana"
	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/plugins/faucet"
)

var (
	// Plugin is the plugin instance of the web API info endpoint plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In

	Server *echo.Echo
	Mesh *mesh_old.Mesh
}

// Plugin gets the plugin instance.
func init() {
	Plugin = node.NewPlugin("WebAPIFaucetEndpoint", deps, node.Disabled, configure)
}

func configure(_ *node.Plugin) {
	deps.Server.POST("faucet", processFaucetRequest)
}

// processFaucetRequest processes the faucet request received via the web API.
func processFaucetRequest(c echo.Context) error {
	var request jsonmodels.FaucetRequest
	if err := c.Bind(&request); err != nil {
		Plugin.LogInfo(err.Error())
		return c.JSON(http.StatusBadRequest, jsonmodels.FaucetAPIResponse{Error: err.Error()})
	}

	Plugin.LogInfo("Received faucet request via web API - address:", request.Address)
	Plugin.LogDebug(request)

	addr, err := devnetvm.AddressFromBase58EncodedString(request.Address)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.FaucetRequestResponse{Error: "Invalid address"})
	}

	var accessManaPledgeID identity.ID
	var consensusManaPledgeID identity.ID
	if request.AccessManaPledgeID == "" {
		return c.JSON(http.StatusBadRequest, jsonmodels.FaucetAPIResponse{Error: "Invalid access mana node ID"})
	}

	if request.ConsensusManaPledgeID == "" {
		return c.JSON(http.StatusBadRequest, jsonmodels.FaucetAPIResponse{Error: "Invalid consensus mana node ID"})
	}

	consensusManaPledgeID, err = mana.IDFromStr(request.ConsensusManaPledgeID)
	accessManaPledgeID, err = mana.IDFromStr(request.AccessManaPledgeID)

	faucetPayload := faucetpkg.NewRequest(addr, accessManaPledgeID, consensusManaPledgeID, request.Nonce)

	err = faucet.OnWebAPIRequest(faucetPayload)

	if err != nil {
		return c.JSON(http.StatusOK, jsonmodels.FaucetAPIResponse{Success: false, Error: err.Error()})
	}
	return c.JSON(http.StatusOK, jsonmodels.FaucetAPIResponse{Success: true})
}

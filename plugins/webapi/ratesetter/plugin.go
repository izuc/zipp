package ratesetter

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/app/blockissuer"
	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/node"
)

// PluginName is the name of the web API info endpoint plugin.
const PluginName = "WebAPIRateSetterEndpoint"

type dependencies struct {
	dig.In

	Server      *echo.Echo
	BlockIssuer *blockissuer.BlockIssuer
}

var (
	// Plugin is the plugin instance of the web API info endpoint plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure)
}

func configure(_ *node.Plugin) {
	deps.Server.GET("ratesetter", getRateSetterEstimate)
}

func getRateSetterEstimate(c echo.Context) error {
	return c.JSON(http.StatusOK, jsonmodels.RateSetter{
		Rate:     deps.BlockIssuer.Rate(),
		Estimate: deps.BlockIssuer.Estimate(),
	})
}

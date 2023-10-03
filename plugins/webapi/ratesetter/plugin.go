package ratesetter

import (
	"net/http"

	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

// PluginName is the name of the web API info endpoint plugin.
const PluginName = "WebAPIRateSetterEndpoint"

type dependencies struct {
	dig.In

	Server *echo.Echo
	Mesh *mesh_old.Mesh
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
		Rate:     deps.Mesh.RateSetter.Rate(),
		Size:     deps.Mesh.RateSetter.Size(),
		Estimate: deps.Mesh.RateSetter.Estimate(),
	})
}

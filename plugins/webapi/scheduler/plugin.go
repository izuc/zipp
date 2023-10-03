package scheduler

import (
	"net/http"

	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

// PluginName is the name of the web API info endpoint plugin.
const PluginName = "WebAPISchedulerEndpoint"

type dependencies struct {
	dig.In

	Server *echo.Echo
	Local  *peer.Local
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
	deps.Server.GET("scheduler", getSchedulerInfo)
}

func getSchedulerInfo(c echo.Context) error {
	nodeQueueSizes := make(map[string]int)
	for nodeID, size := range deps.Mesh.Scheduler.NodeQueueSizes() {
		nodeQueueSizes[nodeID.String()] = size
	}

	deficit, _ := deps.Mesh.Scheduler.GetDeficit(deps.Local.ID()).Float64()
	return c.JSON(http.StatusOK, jsonmodels.Scheduler{
		Running:           deps.Mesh.Scheduler.Running(),
		Rate:              deps.Mesh.Scheduler.Rate().String(),
		MaxBufferSize:     deps.Mesh.Scheduler.MaxBufferSize(),
		CurrentBufferSize: deps.Mesh.Scheduler.BufferSize(),
		NodeQueueSizes:    nodeQueueSizes,
		Deficit:           deficit,
	})
}

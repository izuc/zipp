package data

import (
	"fmt"
	"net/http"
	"time"

	"github.com/izuc/zipp.foundation/core/logger"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/packages/core/mesh_old/payload"
	"github.com/izuc/zipp/plugins/blocklayer"
)

const maxIssuedAwaitTime = 5 * time.Second

// PluginName is the name of the web API data endpoint plugin.
const PluginName = "WebAPIDataEndpoint"

type dependencies struct {
	dig.In

	Server *echo.Echo
	Mesh *mesh_old.Mesh
}

var (
	// Plugin is the plugin instance of the web API data endpoint plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
	log    *logger.Logger
)

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure)
}

func configure(_ *node.Plugin) {
	log = logger.NewLogger(PluginName)
	deps.Server.POST("data", broadcastData)
}

// broadcastData creates a block of the given payload and
// broadcasts it to the node's neighbors. It returns the block ID if successful.
func broadcastData(c echo.Context) error {
	var request jsonmodels.DataRequest
	if err := c.Bind(&request); err != nil {
		log.Info(err.Error())
		return c.JSON(http.StatusBadRequest, jsonmodels.DataResponse{Error: err.Error()})
	}

	if len(request.Data) == 0 {
		return c.JSON(http.StatusBadRequest, jsonmodels.DataResponse{Error: "no data provided"})
	}

	if request.MaxEstimate > 0 && deps.Mesh.RateSetter.Estimate().Milliseconds() > request.MaxEstimate {
		return c.JSON(http.StatusBadRequest, jsonmodels.DataResponse{
			Error: fmt.Sprintf("issuance estimate greater than %d ms", request.MaxEstimate),
		})
	}

	issueData := func() (*mesh_old.Block, error) {
		return deps.Mesh.IssuePayload(payload.NewGenericDataPayload(request.Data))
	}

	// await BlockScheduled event to be triggered.
	blk, err := blocklayer.AwaitBlockToBeIssued(issueData, deps.Mesh.Options.Identity.PublicKey(), maxIssuedAwaitTime)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, jsonmodels.DataResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, jsonmodels.DataResponse{ID: blk.ID().Base58()})
}

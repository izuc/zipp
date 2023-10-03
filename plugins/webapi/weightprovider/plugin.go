package weightprovider

import (
	"net/http"

	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/mesh_old"
)

var (
	// Plugin is the plugin instance of the web API mana endpoint plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In

	Server *echo.Echo
	Mesh *mesh_old.Mesh
}

func init() {
	Plugin = node.NewPlugin("WebAPIWeightProviderEndpoint", deps, node.Enabled, configure)
}

func configure(_ *node.Plugin) {
	deps.Server.GET("weightprovider/activenodes", getNodesHandler)
	deps.Server.GET("weightprovider/weights", getWeightsHandler)
}

func getNodesHandler(c echo.Context) (err error) {
	activeNodes, _ := deps.Mesh.WeightProvider.(*mesh_old.CManaWeightProvider).WeightsOfRelevantVoters()

	activeNodesString := make([]string, 0)

	for id := range activeNodes {
		activeNodesString = append(activeNodesString, id.String())
	}

	return c.JSON(http.StatusOK, activeNodesString)
}

func getWeightsHandler(c echo.Context) (err error) {
	weights, totalWeight := deps.Mesh.WeightProvider.WeightsOfRelevantVoters()

	weightsString := make(map[string]float64)
	for nodeID, mana := range weights {
		weightsString[nodeID.String()] = mana
	}
	resp := Weights{
		Weights:     weightsString,
		TotalWeight: totalWeight,
	}

	return c.JSON(http.StatusOK, resp)
}

// Weights defines the weights associated to the nodes.
type Weights struct {
	Weights     map[string]float64 `json:"weights"`
	TotalWeight float64            `json:"totalWeight"`
}

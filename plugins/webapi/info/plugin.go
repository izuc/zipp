package info

import (
	"net/http"
	"sort"
	"time"

	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58/base58"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/plugins/autopeering/discovery"
	"github.com/izuc/zipp/plugins/banner"
	"github.com/izuc/zipp/plugins/blocklayer"
	"github.com/izuc/zipp/plugins/metrics"
)

// PluginName is the name of the web API info endpoint plugin.
const PluginName = "WebAPIInfoEndpoint"

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
	deps.Server.GET("info", getInfo)
}

// getInfo returns the info of the node
// e.g.,
// {
// 	"version":"v0.2.0",
//	"meshTime":{
// 		"blockID":"24Uq4UFQ7p5oLyjuXX32jHhNreo5hY9eo8Awh36RhdTHCwFMtct3SE2rhe3ceYz6rjKDjBs3usoHS3ujFEabP5ri",
// 		"time":1595528075204868900,
// 		"synced":true
// }
// 	"identityID":"5bf4aa1d6c47e4ce",
// 	"publickey":"CjUsn86jpFHWnSCx3NhWfU4Lk16mDdy1Hr7ERSTv3xn9",
// 	"enabledplugins":[
// 		"Config",
// 		"AutoPeering",
// 		"Analysis",
// 		"WebAPIDataEndpoint",
// 		"BlockLayer",
// 		"CLI",
// 		"Database",
// 		"WebAPIAutoPeeringEndpoint",
// 		"Metrics",
// 		"PortCheck",
// 		"Dashboard",
// 		"WebAPI",
// 		"WebAPIInfoEndpoint",
// 		"WebAPIBlockEndpoint",
// 		"Banner",
// 		"Gossip",
// 		"GracefulShutdown",
// 		"Logger"
// 	],
// 	"disabledplugins":[
// 		"RemoteLog",
// 		"Spammer",
// 		"WebAPIAuth"
// 	]
// }
func getInfo(c echo.Context) error {
	var enabledPlugins []string
	var disabledPlugins []string
	for pluginName, plugin := range node.GetPlugins() {
		if node.IsSkipped(plugin) {
			disabledPlugins = append(disabledPlugins, pluginName)
		} else {
			enabledPlugins = append(enabledPlugins, pluginName)
		}
	}

	sort.Strings(enabledPlugins)
	sort.Strings(disabledPlugins)

	// get MeshTime
	tm := deps.Mesh.TimeManager
	lcm := tm.LastAcceptedBlock()
	meshTime := jsonmodels.MeshTime{
		Synced:          deps.Mesh.TimeManager.Synced(),
		AcceptedBlockID: lcm.BlockID.Base58(),
		ATT:             tm.ATT().UnixNano(),
		RATT:            tm.RATT().UnixNano(),
		CTT:             tm.CTT().UnixNano(),
		RCTT:            tm.RCTT().UnixNano(),
	}

	t := time.Now()
	accessMana, tAccess, _ := blocklayer.GetAccessMana(deps.Local.ID(), t)
	consensusMana, tConsensus, _ := blocklayer.GetConsensusMana(deps.Local.ID(), t)
	nodeMana := jsonmodels.Mana{
		Access:             accessMana,
		AccessTimestamp:    tAccess,
		Consensus:          consensusMana,
		ConsensusTimestamp: tConsensus,
	}

	nodeQueueSizes := make(map[string]int)
	for nodeID, size := range deps.Mesh.Scheduler.NodeQueueSizes() {
		nodeQueueSizes[nodeID.String()] = size
	}

	deficit, _ := deps.Mesh.Scheduler.GetDeficit(deps.Local.ID()).Float64()

	return c.JSON(http.StatusOK, jsonmodels.InfoResponse{
		Version:               banner.AppVersion,
		NetworkVersion:        discovery.Parameters.NetworkVersion,
		MeshTime:            meshTime,
		IdentityID:            base58.Encode(deps.Local.Identity.ID().Bytes()),
		IdentityIDShort:       deps.Local.Identity.ID().String(),
		PublicKey:             deps.Local.PublicKey().String(),
		BlockRequestQueueSize: int(metrics.BlockRequestQueueSize()),
		SolidBlockCount: int(metrics.InitialBlockCountPerComponentGrafana()[metrics.Solidifier] +
			metrics.BlockCountSinceStartPerComponentGrafana()[metrics.Solidifier]),
		TotalBlockCount: int(metrics.InitialBlockCountPerComponentGrafana()[metrics.Store] +
			metrics.BlockCountSinceStartPerComponentGrafana()[metrics.Store]),
		EnabledPlugins:  enabledPlugins,
		DisabledPlugins: disabledPlugins,
		Mana:            nodeMana,
		Scheduler: jsonmodels.Scheduler{
			Running:           deps.Mesh.Scheduler.Running(),
			Rate:              deps.Mesh.Scheduler.Rate().String(),
			MaxBufferSize:     deps.Mesh.Scheduler.MaxBufferSize(),
			CurrentBufferSize: deps.Mesh.Scheduler.BufferSize(),
			Deficit:           deficit,
			NodeQueueSizes:    nodeQueueSizes,
		},
		RateSetter: jsonmodels.RateSetter{
			Rate:     deps.Mesh.RateSetter.Rate(),
			Size:     deps.Mesh.RateSetter.Size(),
			Estimate: deps.Mesh.RateSetter.Estimate(),
		},
	})
}

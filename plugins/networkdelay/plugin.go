package networkdelay

import (
	"sync"
	"time"

	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/crypto/ed25519"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/packages/node/clock"
	"github.com/izuc/zipp/plugins/remotelog"
)

const (
	// PluginName contains the human readable name of the plugin.
	PluginName = "NetworkDelay"

	remoteLogType = "networkdelay"
)

var (
	// App is the "plugin" instance of the network delay application.
	app             *node.Plugin
	once            sync.Once
	deps            = new(dependencies)
	myID            string
	myPublicKey     ed25519.PublicKey
	originPublicKey ed25519.PublicKey

	// clockEnabled defines if the clock plugin is enabled.
	clockEnabled bool
)

type dependencies struct {
	dig.In

	Mesh       *mesh_old.Mesh
	Local        *peer.Local
	RemoteLogger *remotelog.RemoteLoggerConn `optional:"true"`
	Server       *echo.Echo
	ClockPlugin  *node.Plugin `name:"clock"`
}

// App gets the plugin instance.
func App() *node.Plugin {
	once.Do(func() {
		app = node.NewPlugin(PluginName, deps, node.Disabled, configure)
	})
	return app
}

func configure(plugin *node.Plugin) {
	if deps.RemoteLogger == nil {
		plugin.LogInfo("Plugin is inactive since RemoteLogger is disabled")
		return
	}

	if deps.Local != nil {
		myID = deps.Local.ID().String()
		myPublicKey = deps.Local.PublicKey()
	}

	// get origin public key from config
	bytes, err := base58.Decode(Parameters.OriginPublicKey)
	if err != nil {
		plugin.LogFatalfAndExit("could not parse originPublicKey config entry as base58. %v", err)
	}
	originPublicKey, _, err = ed25519.PublicKeyFromBytes(bytes)
	if err != nil {
		plugin.LogFatalfAndExit("could not parse originPublicKey config entry as public key. %v", err)
	}

	configureWebAPI()

	// subscribe to block-layer
	deps.Mesh.ApprovalWeightManager.Events.BlockProcessed.Attach(event.NewClosure(func(event *mesh_old.BlockProcessedEvent) {
		onReceiveBlockFromBlockLayer(event.BlockID)
	}))

	clockEnabled = !node.IsSkipped(deps.ClockPlugin)
}

func onReceiveBlockFromBlockLayer(blockID mesh_old.BlockID) {
	deps.Mesh.Storage.Block(blockID).Consume(func(solidBlock *mesh_old.Block) {
		blockPayload := solidBlock.Payload()
		if blockPayload.Type() != Type {
			return
		}

		// check for node identity
		issuerPubKey := solidBlock.IssuerPublicKey()
		if issuerPubKey != originPublicKey || issuerPubKey == myPublicKey {
			return
		}

		networkDelayObject, ok := blockPayload.(*Payload)
		if !ok {
			app.LogInfo("could not cast payload to network delay payload")

			return
		}

		now := clock.SyncedTime().UnixNano()

		// abort if block was sent more than 1min ago
		// this should only happen due to a node resyncing
		if time.Duration(now-networkDelayObject.SentTime()) > time.Minute {
			app.LogDebugf("Received network delay block with >1min delay\n%s", networkDelayObject)
			return
		}

		sendToRemoteLog(networkDelayObject, now)
	})
}

func sendToRemoteLog(networkDelayObject *Payload, receiveTime int64) {
	m := networkDelay{
		NodeID:      myID,
		ID:          networkDelayObject.ID().String(),
		SentTime:    networkDelayObject.SentTime(),
		ReceiveTime: receiveTime,
		Delta:       receiveTime - networkDelayObject.SentTime(),
		Clock:       clockEnabled,
		Synced:      deps.Mesh.Synced(),
		Type:        remoteLogType,
	}
	_ = deps.RemoteLogger.Send(m)
}

func sendPoWInfo(payload *Payload, powDelta time.Duration) {
	m := networkDelay{
		NodeID:      myID,
		ID:          payload.ID().String(),
		SentTime:    0,
		ReceiveTime: 0,
		Delta:       powDelta.Nanoseconds(),
		Clock:       clockEnabled,
		Synced:      deps.Mesh.Synced(),
		Type:        remoteLogType,
	}
	_ = deps.RemoteLogger.Send(m)
}

type networkDelay struct {
	NodeID      string `json:"nodeId"`
	ID          string `json:"id"`
	SentTime    int64  `json:"sentTime"`
	ReceiveTime int64  `json:"receiveTime"`
	Delta       int64  `json:"delta"`
	Clock       bool   `json:"clock"`
	Synced      bool   `json:"sync"`
	Type        string `json:"type"`
}

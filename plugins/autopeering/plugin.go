package autopeering

import (
	"context"
	"net"
	"time"

	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/app/daemon"
	"github.com/izuc/zipp.foundation/autopeering/discover"
	"github.com/izuc/zipp.foundation/autopeering/peer"
	"github.com/izuc/zipp.foundation/autopeering/peer/service"
	"github.com/izuc/zipp.foundation/autopeering/selection"
	"github.com/izuc/zipp.foundation/autopeering/server"
	"github.com/izuc/zipp.foundation/logger"
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/core/shutdown"
	"github.com/izuc/zipp/packages/network/p2p"
	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/packages/protocol/engine/throughputquota/mana1/manamodels"
	"github.com/izuc/zipp/plugins/autopeering/discovery"
)

// PluginName is the name of the peering plugin.
const PluginName = "AutoPeering"

var (
	// Plugin is the plugin instance of the autopeering plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)

	localAddr *net.UDPAddr
)

type dependencies struct {
	dig.In

	Discovery             *discover.Protocol
	Selection             *selection.Protocol
	Local                 *peer.Local
	P2PMgr                *p2p.Manager                 `optional:"true"`
	ManaFunc              manamodels.ManaRetrievalFunc `optional:"true" name:"manaFunc"`
	AutopeeringConnMetric *UDPConnTraffic
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure, run)

	Plugin.Events.Init.Hook(func(event *node.InitEvent) {
		log := logger.NewLogger(PluginName)

		log.Infof("Initializing %s plugin...", PluginName)

		log.Info("Hooking into Plugin.Events.Init...")

		if err := event.Container.Provide(discovery.CreatePeerDisc); err != nil {
			Plugin.Panic(err)
		}
		log.Info("Provided discovery.CreatePeerDisc to container")

		if err := event.Container.Provide(createPeerSel); err != nil {
			Plugin.Panic(err)
		}
		log.Info("Provided createPeerSel to container")

		if err := event.Container.Provide(func() *node.Plugin {
			return Plugin
		}, dig.Name("autopeering")); err != nil {
			Plugin.Panic(err)
		}
		log.Info("Provided Plugin to container with name 'autopeering'")

		if err := event.Container.Provide(func() *UDPConnTraffic {
			return &UDPConnTraffic{}
		}); err != nil {
			Plugin.Panic(err)
		}
		log.Info("Provided UDPConnTraffic to container")
	})
}

func configure(plugin *node.Plugin) {
	var err error

	_, port, err := net.SplitHostPort(Parameters.BindAddress)
	if err != nil {
		Plugin.LogFatalfAndExitf("failed to parse bind address: %s", err)
	}

	// Construct the new bind address with 0.0.0.0
	bindAddress := "0.0.0.0:" + port

	// Resolve the new bind address
	localAddr, err = net.ResolveUDPAddr("udp4", bindAddress)
	if err != nil {
		Plugin.LogFatalfAndExitf("bind address '%s' is invalid: %s", bindAddress, err)
	} else {
		Plugin.Logger().Infof("Binding to address: %s", bindAddress)
	}

	// announce the peering service
	if err := deps.Local.UpdateService(service.PeeringKey, localAddr.Network(), localAddr.Port); err != nil {
		Plugin.LogFatalfAndExitf("could not update services: %s", err)
	}

	if deps.P2PMgr != nil {
		configureGossipIntegration(plugin)
	}
	configureEvents(plugin)
}

func run(*node.Plugin) {
	if err := daemon.BackgroundWorker(PluginName, start, shutdown.PriorityAutopeering); err != nil {
		Plugin.Panicf("Failed to start as daemon: %s", err)
	}
}

func configureGossipIntegration(plugin *node.Plugin) {
	// assure that the Manager is instantiated
	mgr := deps.P2PMgr

	// link to the autopeering events
	deps.Selection.Events().Dropped.Hook(func(ev *selection.DroppedEvent) {
		if err := mgr.DropNeighbor(ev.DroppedID, p2p.NeighborsGroupAuto); err != nil {
			Plugin.Logger().Debugw("error dropping neighbor", "id", ev.DroppedID, "err", err)
		}
	}, event.WithWorkerPool(plugin.WorkerPool))
	// We need to allocate synchronously the resources to accommodate incoming stream requests.
	deps.Selection.Events().IncomingPeering.Hook(func(ev *selection.PeeringEvent) {
		if !ev.Status {
			return // ignore rejected peering
		}
		if err := mgr.AddInbound(context.Background(), ev.Peer, p2p.NeighborsGroupAuto); err != nil {
			deps.Selection.RemoveNeighbor(ev.Peer.ID())
			Plugin.Logger().Debugw("error adding inbound", "id", ev.Peer.ID(), "err", err)
		}
	})

	deps.Selection.Events().OutgoingPeering.Hook(func(ev *selection.PeeringEvent) {
		if !ev.Status {
			return // ignore rejected peering
		}
		if err := mgr.AddOutbound(context.Background(), ev.Peer, p2p.NeighborsGroupAuto); err != nil {
			deps.Selection.RemoveNeighbor(ev.Peer.ID())
			Plugin.Logger().Debugw("error adding outbound", "id", ev.Peer.ID(), "err", err)
		}
	}, event.WithWorkerPool(plugin.WorkerPool))

	mgr.NeighborGroupEvents(p2p.NeighborsGroupAuto).NeighborRemoved.Hook(func(event *p2p.NeighborRemovedEvent) {
		deps.Selection.RemoveNeighbor(event.Neighbor.ID())
	}, event.WithWorkerPool(plugin.WorkerPool))
}

func configureEvents(plugin *node.Plugin) {
	// log the peer discovery events
	deps.Discovery.Events().PeerDiscovered.Hook(func(ev *discover.PeerDiscoveredEvent) {
		Plugin.Logger().Infof("Discovered: %s / %s", ev.Peer.Address(), ev.Peer.ID())
		Plugin.Logger().Debugf("Discovered details: %v", ev.Peer)
	}, event.WithWorkerPool(plugin.WorkerPool))
	deps.Discovery.Events().PeerDeleted.Hook(func(ev *discover.PeerDeletedEvent) {
		Plugin.Logger().Infof("Removed offline: %s / %s", ev.Peer.Address(), ev.Peer.ID())
	}, event.WithWorkerPool(plugin.WorkerPool))

	// log the peer selection events
	deps.Selection.Events().SaltUpdated.Hook(func(ev *selection.SaltUpdatedEvent) {
		Plugin.Logger().Infof("Salt updated; expires=%s", ev.Public.GetExpiration().Format(time.RFC822))
	}, event.WithWorkerPool(plugin.WorkerPool))
	deps.Selection.Events().OutgoingPeering.Hook(func(ev *selection.PeeringEvent) {
		if ev.Status {
			Plugin.Logger().Infof("Peering chosen: %s / %s", ev.Peer.Address(), ev.Peer.ID())
		} else {
			Plugin.Logger().Warnf("Peering attempt failed: %s / %s", ev.Peer.Address(), ev.Peer.ID())
		}
	}, event.WithWorkerPool(plugin.WorkerPool))
	deps.Selection.Events().IncomingPeering.Hook(func(ev *selection.PeeringEvent) {
		if ev.Status {
			Plugin.Logger().Infof("Peering accepted: %s / %s", ev.Peer.Address(), ev.Peer.ID())
		} else {
			Plugin.Logger().Warnf("Peering rejected: %s / %s", ev.Peer.Address(), ev.Peer.ID())
		}
	}, event.WithWorkerPool(plugin.WorkerPool))
	deps.Selection.Events().Dropped.Hook(func(ev *selection.DroppedEvent) {
		Plugin.Logger().Infof("Peering dropped: %s", ev.DroppedID)
	}, event.WithWorkerPool(plugin.WorkerPool))
}

func start(ctx context.Context) {
	defer Plugin.Logger().Info("Stopping " + PluginName + " ... done")

	conn, err := net.ListenUDP("udp4", localAddr)
	if err != nil {
		Plugin.Logger().Fatalf("Error listening: %v", err)
	} else {
		Plugin.Logger().Debugf("Started UDP listener on: %s", localAddr.String()) // ADDED
	}
	defer conn.Close()

	// use wrapped UDPConn to allow metrics collection
	deps.AutopeeringConnMetric.UDPConn = conn

	lPeer := deps.Local

	// start a server doing peerDisc and peering
	srv := server.Serve(lPeer, deps.AutopeeringConnMetric, Plugin.Logger().Named("srv"), deps.Discovery, deps.Selection)
	defer srv.Close()

	// log before starting the protocols
	Plugin.Logger().Debug("Starting Discovery protocol...")
	deps.Discovery.Start(srv)
	Plugin.Logger().Debug("Discovery protocol started.")

	Plugin.Logger().Debug("Starting Selection protocol...")
	deps.Selection.Start(srv)
	Plugin.Logger().Debug("Selection protocol started.")

	Plugin.Logger().Infof("%s started: ID=%s Address=%s/%s", PluginName, lPeer.ID(), localAddr.String(), localAddr.Network())

	<-ctx.Done()

	Plugin.Logger().Infof("Stopping %s ...", PluginName)
	deps.Selection.Close()

	deps.Discovery.Close()

	lPeer.Database().Close()
}

package chat

import (
	"github.com/labstack/echo"
	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/packages/app/chat"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

const (
	// PluginName contains the human-readable name of the plugin.
	PluginName = "Chat"
)

var (
	// Plugin is the "plugin" instance of the chat application.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure)
	Plugin.Events.Init.Hook(event.NewClosure[*node.InitEvent](func(event *node.InitEvent) {
		if err := event.Container.Provide(chat.NewChat); err != nil {
			Plugin.Panic(err)
		}
	}))
}

type dependencies struct {
	dig.In
	Mesh *mesh_old.Mesh
	Server *echo.Echo
	Chat   *chat.Chat
}

func configure(_ *node.Plugin) {
	deps.Mesh.Booker.Events.BlockBooked.Attach(event.NewClosure(func(event *mesh_old.BlockBookedEvent) {
		onReceiveBlockFromBlockLayer(event.BlockID)
	}))
	configureWebAPI()
}

func onReceiveBlockFromBlockLayer(blockID mesh_old.BlockID) {
	var chatEvent *chat.BlockReceivedEvent
	deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
		if block.Payload().Type() != chat.Type {
			return
		}
		chatPayload := block.Payload().(*chat.Payload)
		chatEvent = &chat.BlockReceivedEvent{
			From:      chatPayload.From(),
			To:        chatPayload.To(),
			Block:     chatPayload.Block(),
			Timestamp: block.IssuingTime(),
			BlockID:   block.ID().Base58(),
		}
	})

	if chatEvent == nil {
		return
	}

	deps.Chat.Events.BlockReceived.Trigger(chatEvent)
}

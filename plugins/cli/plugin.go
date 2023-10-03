package cli

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/plugins/banner"
)

// PluginName is the name of the CLI plugin.
const PluginName = "CLI"

var (
	// Plugin is the plugin instance of the CLI plugin.
	Plugin  *node.Plugin
	version = flag.BoolP("version", "v", false, "Prints the ZIPP version")
)

func init() {
	Plugin = node.NewPlugin(PluginName, nil, node.Enabled)

	for name, plugin := range node.GetPlugins() {
		onAddPlugin(&node.AddEvent{Name: name, Status: plugin.Status})
	}

	node.Events.AddPlugin.Hook(event.NewClosure(onAddPlugin))

	flag.Usage = printUsage

	Plugin.Events.Init.Hook(event.NewClosure(onInit))
}

func onAddPlugin(addEvent *node.AddEvent) {
	AddPluginStatus(node.GetPluginIdentifier(addEvent.Name), addEvent.Status)
}

func onInit(_ *node.InitEvent) {
	if *version {
		fmt.Println(banner.AppName + " " + banner.AppVersion)
		os.Exit(0)
	}
}

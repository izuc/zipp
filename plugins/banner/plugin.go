package banner

import (
	"fmt"
	"strings"

	"github.com/izuc/zipp.foundation/core/node"
)

// PluginName is the name of the banner plugin.
const PluginName = "Banner"

var (
	// Plugin is the plugin instance of the banner plugin.
	Plugin = node.NewPlugin(PluginName, nil, node.Enabled, configure, run)

	// AppVersion version number
	AppVersion = "v0.1.3"
	// SimplifiedAppVersion is the version number without commit hash
	SimplifiedAppVersion = simplifiedVersion(AppVersion)
)

const (
	// AppName app code name
	AppName = "ZIPP"
)

func configure(ctx *node.Plugin) {
	fmt.Printf(`
	____  _  ____  ____ 
	/_   \/ \/  __\/  __\
	 /   /| ||  \/||  \/|
	/   /_| ||  __/|  __/
	\____/\_/\_/   \_/   						 	
                             %s                                     
`, AppVersion)
	fmt.Println()

	ctx.Node.Logger.Infof("ZIPP version %s ...", AppVersion)
	ctx.Node.Logger.Info("Loading plugins ...")
}

func run(ctx *node.Plugin) {}

func simplifiedVersion(version string) string {
	// ignore commit hash
	ver := version
	if strings.Contains(ver, "-") {
		ver = strings.Split(ver, "-")[0]
	}
	// attach a "v" at front
	if !strings.Contains(ver, "v") {
		ver = "v" + ver
	}
	return ver
}

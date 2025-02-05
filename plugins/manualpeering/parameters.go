package manualpeering

import "github.com/izuc/zipp/plugins/config"

// ParametersDefinition contains the definition of the parameters used by the manualPeering plugin.
type ParametersDefinition struct {
	// KnownPeers defines the map of peers to be used as known peers.
	KnownPeers string `usage:"map of peers that will be used as known peers"`
}

// Parameters contains the configuration used by the manualPeering plugin.
var Parameters = &ParametersDefinition{}

func init() {
	config.BindParameters(Parameters, "manualPeering")
}

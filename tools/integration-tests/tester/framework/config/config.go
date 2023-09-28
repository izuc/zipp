package config

import (
	"reflect"

	"github.com/izuc/zipp.foundation/crypto/ed25519"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp/plugins/activity"
	"github.com/izuc/zipp/plugins/autopeering"
	"github.com/izuc/zipp/plugins/autopeering/discovery"
	"github.com/izuc/zipp/plugins/blockissuer"
	"github.com/izuc/zipp/plugins/dagsvisualizer"
	"github.com/izuc/zipp/plugins/dashboard"
	"github.com/izuc/zipp/plugins/faucet"
	"github.com/izuc/zipp/plugins/metrics"
	"github.com/izuc/zipp/plugins/p2p"
	"github.com/izuc/zipp/plugins/profiling"
	"github.com/izuc/zipp/plugins/protocol"
	"github.com/izuc/zipp/plugins/webapi"
)

// ZIPP defines the config of a ZIPP node.
type ZIPP struct {
	// Name specifies the ZIPP instance.
	Name string
	// Image specifies the docker image for the instance
	Image string
	// DisabledPlugins specifies the plugins that are disabled with a config.
	DisabledPlugins []string
	// Seed specifies identity.
	Seed []byte
	// Whether to use the same seed for the node's wallet.
	UseNodeSeedAsWalletSeed bool

	// Network specifies network-level configurations
	Network

	// individual plugin configurations
	Database
	P2P
	WebAPI
	BlockIssuer
	AutoPeering
	Protocol
	Faucet
	Activity
	Metrics
	Profiling
	Dashboard
	Dagsvisualizer
	Notarization
}

// NewZIPP creates a ZIPP config initialized with default values.
func NewZIPP() (config ZIPP) {
	config = ZIPP{}
	fillStructFromDefaultTag(reflect.ValueOf(&config).Elem())
	return
}

type Network struct {
	Enabled bool
}

// Database defines the parameters of the database plugin.
type Database struct {
	Enabled bool

	protocol.DatabaseParametersDefinition
}

// P2P defines the parameters of the gossip plugin.
type P2P struct {
	Enabled bool

	p2p.ParametersDefinition
}

// WebAPI defines the parameters of the Web API plugin.
type WebAPI struct {
	Enabled bool

	webapi.ParametersDefinition
}

// AutoPeering defines the parameters of the autopeering plugin.
type AutoPeering struct {
	Enabled bool

	autopeering.ParametersDefinition
	discovery.ParametersDefinitionDiscovery
}

// Faucet defines the parameters of the faucet plugin.
type Faucet struct {
	Enabled bool

	faucet.ParametersDefinition
}

// Protocol defines the parameters used by the block layer.
type Protocol struct {
	Enabled bool

	protocol.ParametersDefinition
}

// Activity defines the parameters of the activity plugin.
type Activity struct {
	Enabled bool

	activity.ParametersDefinition
}

// BlockIssuer defines the parameters of the BlockIssuer plugin.
type BlockIssuer struct {
	Enabled bool

	blockissuer.ParametersDefinition
}

// Metrics defines the parameters of the Metrics plugin.
type Metrics struct {
	Enabled bool

	metrics.ParametersDefinition
}

// Profiling defines the parameters of the Profiling plugin.
type Profiling struct {
	Enabled bool

	profiling.ParametersDefinition
}

// Dashboard defines the parameters of the Dashboard plugin.
type Dashboard struct {
	Enabled bool

	dashboard.ParametersDefinition
}

// Dagsvisualizer defines the parameters of the Dag Visualizer plugin.
type Dagsvisualizer struct {
	Enabled bool

	dagsvisualizer.ParametersDefinition
}

// Notarization defines the parameters of the Notarization plugin.
type Notarization struct {
	Enabled bool

	protocol.NotarizationParametersDefinition
}

// CreateIdentity returns an identity based on the config.
// If a Seed is specified, it is used to derive the identity. Otherwise a new key pair is generated and Seed set accordingly.
func (s *ZIPP) CreateIdentity() (*identity.Identity, error) {
	if s.Seed != nil {
		publicKey := ed25519.PrivateKeyFromSeed(s.Seed).Public()
		return identity.New(publicKey), nil
	}

	publicKey, privateKey, err := ed25519.GenerateKey()
	if err != nil {
		return nil, err
	}
	s.Seed = privateKey.Seed().Bytes()
	return identity.New(publicKey), nil
}

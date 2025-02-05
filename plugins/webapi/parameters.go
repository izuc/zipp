package webapi

import "github.com/izuc/zipp/plugins/config"

// ParametersDefinition contains the definition of the parameters used by the webAPI plugin.
type ParametersDefinition struct {
	// BindAddress defines the bind address for the web API.
	BindAddress string `default:"127.0.0.1:8080" usage:"the bind address for the web API"`

	// BasicAuth
	BasicAuth struct {
		// Enabled defines whether basic HTTP authentication is required to access the API.
		Enabled bool `default:"false" usage:"whether to enable HTTP basic auth"`
		// Username defines the user used by the basic HTTP authentication.
		Username string `default:"zipp" usage:"HTTP basic auth username"`
		// Password defines the password used by the basic HTTP authentication.
		Password string `default:"zipp" usage:"HTTP basic auth password"`
	}

	// EnableDSFilter determines if the DoubleSpendFilter should be enabled.
	EnableDSFilter bool `default:"false" usage:"whether to enable double spend filter"`
}

// Parameters contains the configuration used by the webAPI plugin.
var Parameters = &ParametersDefinition{}

func init() {
	config.BindParameters(Parameters, "webAPI")
}

package config

type ZIPPPort int

const (
	WebApiPort    ZIPPPort = 8080
	DashboardPort ZIPPPort = 8081
	DagVizPort    ZIPPPort = 8061
	DebugPort     ZIPPPort = 40000
)

// ZIPPPorts is the collection of ports that should be exposed by socat
var ZIPPPorts = []ZIPPPort{
	WebApiPort,
	DashboardPort,
	DagVizPort,
	DebugPort,
}

package client

import (
	"net/http"
)

const (
	routeHealth = "healthz"
)

// HealthCheck checks whether the node is running and healthy.
func (api *ZIPPAPI) HealthCheck() error {
	return api.do(http.MethodGet, routeHealth, nil, nil)
}

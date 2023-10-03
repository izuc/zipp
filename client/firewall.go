package client

import (
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/izuc/zipp.foundation/core/identity"
)

const (
	routeFirewallPeerFaultinessCount = "firewall/peer-faultiness-count"
)

// GetPeerFaultinessCount return number of time peer has been marked as faulty.
func (api *ZIPPAPI) GetPeerFaultinessCount(peerID identity.ID) (int, error) {
	var count int
	if err := api.do(
		http.MethodGet,
		fmt.Sprintf("%s/%s", routeFirewallPeerFaultinessCount, peerID.EncodeBase58()),
		nil, &count,
	); err != nil {
		return 0, errors.Wrap(err, "failed to fetch peer faultiness details via HTTP API")
	}
	return count, nil
}

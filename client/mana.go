package client

import (
	"fmt"
	"net/http"

	"github.com/izuc/zipp/packages/app/jsonmodels"
)

const (
	routeGetMana                  = "mana"
	routeGetAllMana               = "mana/all"
	routeGetManaPercentile        = "mana/percentile"
	routeGetOnlineAccessMana      = "mana/access/online"
	routeGetOnlineConsensusMana   = "mana/consensus/online"
	routeGetNHighestAccessMana    = "mana/access/nhighest"
	routeGetNHighestConsensusMana = "mana/consensus/nhighest"
	routePending                  = "mana/pending"
	routePastConsensusEventLogs   = "mana/consensus/logs"
	routeAllowedPledgeNodeIDs     = "mana/allowedManaPledge"
)

// GetOwnMana returns the access and consensus mana of the node this api client is communicating with.
func (api *ZIPPAPI) GetOwnMana() (*jsonmodels.GetManaResponse, error) {
	res := &jsonmodels.GetManaResponse{}
	if err := api.do(http.MethodGet, routeGetMana,
		&jsonmodels.GetManaRequest{NodeID: ""}, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetManaFullNodeID returns the access and consensus mana of the node specified in the argument.
// Note, that for the node to understand which nodeID we are referring to, short node ID is not sufficient.
func (api *ZIPPAPI) GetManaFullNodeID(fullNodeID string) (*jsonmodels.GetManaResponse, error) {
	res := &jsonmodels.GetManaResponse{}
	if err := api.do(http.MethodGet, routeGetMana,
		&jsonmodels.GetManaRequest{NodeID: fullNodeID}, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetMana returns the access and consensus mana a node has based on its shortNodeID.
func (api *ZIPPAPI) GetMana(shortNodeID string) (*jsonmodels.GetManaResponse, error) {
	// ask the node about the full mana map and filter out based on shortID
	allManaRes := &jsonmodels.GetAllManaResponse{}
	if err := api.do(http.MethodGet, routeGetAllMana,
		nil, allManaRes); err != nil {
		return nil, err
	}
	res := &jsonmodels.GetManaResponse{ShortNodeID: shortNodeID}
	// look for node's mana values in the map
	for _, nodeStr := range allManaRes.Access {
		if nodeStr.ShortNodeID == shortNodeID {
			res.Access = nodeStr.Mana
			break
		}
	}
	for _, nodeStr := range allManaRes.Consensus {
		if nodeStr.ShortNodeID == shortNodeID {
			res.Consensus = nodeStr.Mana
			break
		}
	}
	return res, nil
}

// GetAllMana returns the mana perception of the node in the network.
func (api *ZIPPAPI) GetAllMana() (*jsonmodels.GetAllManaResponse, error) {
	res := &jsonmodels.GetAllManaResponse{}
	if err := api.do(http.MethodGet, routeGetAllMana,
		nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetManaPercentile returns the mana percentile for access and consensus mana of a node.
func (api *ZIPPAPI) GetManaPercentile(fullNodeID string) (*jsonmodels.GetPercentileResponse, error) {
	res := &jsonmodels.GetPercentileResponse{}
	if err := api.do(http.MethodGet, routeGetManaPercentile,
		&jsonmodels.GetPercentileRequest{NodeID: fullNodeID}, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetOnlineAccessMana returns the sorted list of online access mana of nodes.
func (api *ZIPPAPI) GetOnlineAccessMana() (*jsonmodels.GetOnlineResponse, error) {
	res := &jsonmodels.GetOnlineResponse{}
	if err := api.do(http.MethodGet, routeGetOnlineAccessMana,
		nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetOnlineConsensusMana returns the sorted list of online consensus mana of nodes.
func (api *ZIPPAPI) GetOnlineConsensusMana() (*jsonmodels.GetOnlineResponse, error) {
	res := &jsonmodels.GetOnlineResponse{}
	if err := api.do(http.MethodGet, routeGetOnlineConsensusMana,
		nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetNHighestAccessMana returns the N highest access mana holders in the network, sorted in descending order.
func (api *ZIPPAPI) GetNHighestAccessMana(n int) (*jsonmodels.GetNHighestResponse, error) {
	res := &jsonmodels.GetNHighestResponse{}
	if err := api.do(http.MethodGet, func() string {
		return fmt.Sprintf("%s?number=%d", routeGetNHighestAccessMana, n)
	}(), nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetNHighestConsensusMana returns the N highest consensus mana holders in the network, sorted in descending order.
func (api *ZIPPAPI) GetNHighestConsensusMana(n int) (*jsonmodels.GetNHighestResponse, error) {
	res := &jsonmodels.GetNHighestResponse{}
	if err := api.do(http.MethodGet, func() string {
		return fmt.Sprintf("%s?number=%d", routeGetNHighestConsensusMana, n)
	}(), nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetConsensusEventLogs returns the consensus event logs or the nodeIDs specified.
func (api *ZIPPAPI) GetConsensusEventLogs(nodeIDs []string) (*jsonmodels.GetEventLogsResponse, error) {
	res := &jsonmodels.GetEventLogsResponse{}
	if err := api.do(http.MethodGet, routePastConsensusEventLogs,
		&jsonmodels.GetEventLogsRequest{NodeIDs: nodeIDs}, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetAllowedManaPledgeNodeIDs returns the list of allowed mana pledge IDs.
func (api *ZIPPAPI) GetAllowedManaPledgeNodeIDs() (*jsonmodels.AllowedManaPledgeResponse, error) {
	res := &jsonmodels.AllowedManaPledgeResponse{}
	if err := api.do(http.MethodGet, routeAllowedPledgeNodeIDs, nil, res); err != nil {
		return nil, err
	}

	return res, nil
}

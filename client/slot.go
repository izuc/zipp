package client

import (
	"net/http"
	"strconv"

	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp/packages/app/jsonmodels"
)

const (
	routeLatestSlotCommitment = "sc"
	routeCommitment           = "slots/"
	routeLatestConfirmedIndex = "latest-confirmed-index"
)

func (api *GoShimmerAPI) GetLatestCommittedSlotInfo() (*jsonmodels.SlotInfo, error) {
	res := &jsonmodels.SlotInfo{}
	if err := api.do(http.MethodGet, routeLatestSlotCommitment, nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (api *GoShimmerAPI) GetSlotInfo(epochIndex int) (*jsonmodels.SlotInfo, error) {
	res := &jsonmodels.SlotInfo{}
	if err := api.do(
		http.MethodGet,
		routeCommitment+strconv.Itoa(epochIndex),
		nil,
		res,
	); err != nil {
		return nil, err
	}
	return res, nil
}

func (api *GoShimmerAPI) GetLatestConfirmedIndex() (slot.Index, error) {
	res := &jsonmodels.LatestConfirmedIndexResponse{}
	if err := api.do(http.MethodGet, routeLatestConfirmedIndex, nil, res); err != nil {
		return 0, err
	}
	return slot.Index(res.Index), nil
}

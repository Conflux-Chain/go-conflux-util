package metrics

import (
	"strings"

	"github.com/ethereum/go-ethereum/metrics"
)

const DefaultAPIName = "metrics"

// API provides metrics related RPC implementations.
type API struct {
	reg metrics.Registry
}

func NewAPI(reg metrics.Registry) *API { return &API{reg} }

func NewDefaultAPI() *API { return &API{DefaultRegistry} }

// GetMetrics returns all metrics of specified prefix. Empty prefix indicates all metrics.
func (api *API) GetMetrics(prefix ...string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	all := api.reg.GetAll()

	for k, v := range all {
		if len(prefix) == 0 || strings.HasPrefix(k, prefix[0]) {
			result[k] = v
		}
	}

	return result, nil
}

package metrics

import (
	"math"
	"strings"

	"github.com/rcrowley/go-metrics"
)

const DefaultAPIName = "metrics"

// API provides metrics related RPC implementations.
type API struct {
	reg metrics.Registry
}

func NewAPI(reg metrics.Registry) *API { return &API{reg} }

func NewDefaultAPI() *API { return &API{metrics.DefaultRegistry} }

// GetMetrics returns all metrics of specified prefix. Empty prefix indicates all metrics.
// Non-finite float64 field values (NaN and +/-Inf) are returned as nil, encoded as JSON null.
func (api *API) GetMetrics(prefix ...string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	all := api.reg.GetAll()

	for k, v := range all {
		if len(prefix) == 0 || strings.HasPrefix(k, prefix[0]) {
			for field, value := range v {
				if number, ok := value.(float64); ok && (math.IsNaN(number) || math.IsInf(number, 0)) {
					v[field] = nil
				}
			}
			result[k] = v
		}
	}

	return result, nil
}

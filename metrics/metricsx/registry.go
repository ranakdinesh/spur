//go:build metrics

package metricsx

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	r *prometheus.Registry
}

func NewRegistry() *Registry {
	r := prometheus.NewRegistry()
	// Standard collectors
	r.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	return &Registry{r: r}
}

func (m *Registry) Handler() http.Handler { return promhttp.HandlerFor(m.r, promhttp.HandlerOpts{}) }
func (m *Registry) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		m.r.MustRegister(c)
	}
}
func (m *Registry) Registry() *prometheus.Registry { return m.r }

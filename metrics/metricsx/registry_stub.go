//go:build !metrics

package metricsx

type dummy struct{}

func NewGauge(_ *Registry, _, _, _, _ string, _ map[string]string) any   { return dummy{} }
func NewCounter(_ *Registry, _, _, _, _ string, _ map[string]string) any { return dummy{} }

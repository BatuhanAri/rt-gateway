package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	reg *prometheus.Registry

	ConnectionsCurrent  prometheus.Gauge
	ConnectionsAccepted prometheus.Counter
	MessagesIn          prometheus.Counter
	MessagesOut         prometheus.Counter
	Disconnects         prometheus.Counter
}

func New() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		reg: reg,
		ConnectionsCurrent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rtgateway_connections_current",
			Help: "Current number of open WebSocket connections.",
		}),
		ConnectionsAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rtgateway_connections_accepted_total",
			Help: "Total number of accepted WebSocket connections.",
		}),
		MessagesIn: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rtgateway_messages_in_total",
			Help: "Total number of incoming messages.",
		}),
		MessagesOut: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rtgateway_messages_out_total",
			Help: "Total number of outgoing messages.",
		}),
		Disconnects: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rtgateway_disconnects_total",
			Help: "Total number of disconnects (any reason).",
		}),
	}

	reg.MustRegister(
		m.ConnectionsCurrent,
		m.ConnectionsAccepted,
		m.MessagesIn,
		m.MessagesOut,
		m.Disconnects,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

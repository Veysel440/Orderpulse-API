package stream

import "github.com/prometheus/client_golang/prometheus"

var (
	subsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "stream_subscribers",
		Help: "active subscribers",
	})
	dropsCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "stream_dropped_messages_total",
		Help: "messages dropped due to backpressure",
	})
)

func init() { prometheus.MustRegister(subsGauge, dropsCtr) }

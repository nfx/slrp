package prometheus

import (
	"github.com/nfx/slrp/app"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Prometheus struct {
	name string `json:",omitempty"` // TODO: this is a hack?...
}

func NewPrometheus() *Prometheus {
	return &Prometheus{
		name: "Prometheus",
	}
}

func (p *Prometheus) Start(ctx app.Context) {
	go p.main(ctx)
}

func (p *Prometheus) main(ctx app.Context) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		return
	}
}

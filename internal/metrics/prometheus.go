package metrics

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	Gauge instrument = iota
	Counter
	Histogram
	Summary
)

var (
	ErrInstrumentNotSupported = errors.New("instrument type unsupported")
)

// abstract prometheus types
type instrument int

// combine all possible prometheus options + way to differentiate between regular or vector type
type instrumentOpts struct {
	opts               prometheus.Opts
	buckets            []float64
	objectives         map[float64]float64
	maxAge             time.Duration
	ageBuckets, bufCap uint32
	vectors            []string
}

// InstrumentOption - vararg for instrument options setting
type InstrumentOption func(o *instrumentOpts)

// Vectors - configuration used to create a vector of a given interface, slice of label names
func Vectors(labels []string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.vectors = labels
	}
}

// Help - set the help field on instrument
func Help(help string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.Help = help
	}
}

// Namespace - set namespace
func Namespace(ns string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.Namespace = ns
	}
}

// Subsystem - set subsystem... obviously
func Subsystem(s string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.Subsystem = s
	}
}

// Set labels for instrument (similar to vector, but with given values)
func Labels(labels map[string]string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.ConstLabels = prometheus.Labels(labels)
	}
}

// Buckets - specific to histogram type
func Buckets(b []float64) InstrumentOption {
	return func(o *instrumentOpts) {
		o.buckets = b
	}
}

// Objectives - specific to summary type
func Objectives(obj map[float64]float64) InstrumentOption {
	return func(o *instrumentOpts) {
		o.objectives = obj
	}
}

// MaxAge - specific to summary type
func MaxAge(m time.Duration) InstrumentOption {
	return func(o *instrumentOpts) {
		o.maxAge = m
	}
}

// AgeBuckets - specific to summary type
func AgeBuckets(ab uint32) InstrumentOption {
	return func(o *instrumentOpts) {
		o.ageBuckets = ab
	}
}

// BufCap - specific to summary type
func BufCap(bc uint32) InstrumentOption {
	return func(o *instrumentOpts) {
		o.bufCap = bc
	}
}

// AddInstrument, configure and register new metrics instrument
// this will, over time, be moved to use custom Registries, etc...
func AddInstrument(t instrument, name string, opts ...InstrumentOption) error {
	var col prometheus.Collector
	opt := instrumentOpts{
		opts: prometheus.Opts{
			Name: name,
		},
	}
	// apply options
	for _, o := range opts {
		o(&opt)
	}
	switch t {
	case Gauge:
		o := opt.gauge()
		if len(opt.vectors) == 0 {
			col = prometheus.NewGauge(o)
		} else {
			col = prometheus.NewGaugeVec(o, opt.vectors)
		}
	case Counter:
		o := opt.counter()
		if len(opt.vectors) == 0 {
			col = prometheus.NewCounter(o)
		} else {
			col = prometheus.NewCounterVec(o, opt.vectors)
		}
	case Histogram:
		o := opt.histogram()
		if len(opt.vectors) == 0 {
			col = prometheus.NewHistogram(o)
		} else {
			col = prometheus.NewHistogramVec(o, opt.vectors)
		}
	case Summary:
		o := opt.summary()
		if len(opt.vectors) == 0 {
			col = prometheus.NewSummary(o)
		} else {
			col = prometheus.NewSummaryVec(o, opt.vectors)
		}
	default:
		return ErrInstrumentNotSupported
	}
	return prometheus.Register(col)
}

// Start enable metrics (given config)
func Start(conf Config) {
	if !conf.Enabled {
		return
	}
	http.Handle(conf.Path, promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil))
	}()
}

func (i instrumentOpts) gauge() prometheus.GaugeOpts {
	return prometheus.GaugeOpts(i.opts)
}

func (i instrumentOpts) counter() prometheus.CounterOpts {
	return prometheus.CounterOpts(i.opts)
}

func (i instrumentOpts) summary() prometheus.SummaryOpts {
	return prometheus.SummaryOpts{
		Name:        i.opts.Name,
		Namespace:   i.opts.Namespace,
		Subsystem:   i.opts.Subsystem,
		ConstLabels: i.opts.ConstLabels,
		Help:        i.opts.Help,
		Objectives:  i.objectives,
		MaxAge:      i.maxAge,
		AgeBuckets:  i.ageBuckets,
		BufCap:      i.bufCap,
	}
}

func (i instrumentOpts) histogram() prometheus.HistogramOpts {
	return prometheus.HistogramOpts{
		Name:        i.opts.Name,
		Namespace:   i.opts.Namespace,
		Subsystem:   i.opts.Subsystem,
		ConstLabels: i.opts.ConstLabels,
		Help:        i.opts.Help,
		Buckets:     i.buckets,
	}
}

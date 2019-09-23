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
	ErrInstrumentTypeMismatch = errors.New("instrument is not of the expected type")
)

var (
	unconfirmedTxGauge prometheus.Gauge
	engineTime         *prometheus.CounterVec
	orderCounter       *prometheus.CounterVec
	orderGauge         *prometheus.GaugeVec
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

type mi struct {
	gaugeV     *prometheus.GaugeVec
	gauge      prometheus.Gauge
	counterV   *prometheus.CounterVec
	counter    prometheus.Counter
	histogramV *prometheus.HistogramVec
	histogram  prometheus.Histogram
	summaryV   *prometheus.SummaryVec
	summary    prometheus.Summary
}

// MetricInstrument - template interface for mi type return value - only mock if needed, and only mock the funcs you use
type MetricInstrument interface {
	Gauge() (prometheus.Gauge, error)
	GaugeVec() (*prometheus.GaugeVec, error)
	Counter() (prometheus.Counter, error)
	CounterVec() (*prometheus.CounterVec, error)
	Histogram() (prometheus.Histogram, error)
	HistogramVec() (*prometheus.HistogramVec, error)
	Summary() (prometheus.Summary, error)
	SummaryVec() (*prometheus.SummaryVec, error)
}

// InstrumentOption - vararg for instrument options setting
type InstrumentOption func(o *instrumentOpts)

// Vectors - configuration used to create a vector of a given interface, slice of label names
func Vectors(labels ...string) InstrumentOption {
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
func AddInstrument(t instrument, name string, opts ...InstrumentOption) (*mi, error) {
	var col prometheus.Collector
	ret := mi{}
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
			ret.gauge = prometheus.NewGauge(o)
			col = ret.gauge
		} else {
			ret.gaugeV = prometheus.NewGaugeVec(o, opt.vectors)
			col = ret.gaugeV
		}
	case Counter:
		o := opt.counter()
		if len(opt.vectors) == 0 {
			ret.counter = prometheus.NewCounter(o)
			col = ret.counter
		} else {
			ret.counterV = prometheus.NewCounterVec(o, opt.vectors)
			col = ret.counterV
		}
	case Histogram:
		o := opt.histogram()
		if len(opt.vectors) == 0 {
			ret.histogram = prometheus.NewHistogram(o)
			col = ret.histogram
		} else {
			ret.histogramV = prometheus.NewHistogramVec(o, opt.vectors)
			col = ret.histogramV
		}
	case Summary:
		o := opt.summary()
		if len(opt.vectors) == 0 {
			ret.summary = prometheus.NewSummary(o)
			col = ret.summary
		} else {
			ret.summaryV = prometheus.NewSummaryVec(o, opt.vectors)
			col = ret.summaryV
		}
	default:
		return nil, ErrInstrumentNotSupported
	}
	if err := prometheus.Register(col); err != nil {
		return nil, err
	}
	return &ret, nil
}

// Start enable metrics (given config)
func Start(conf Config) {
	if !conf.Enabled {
		return
	}
	err := setupMetrics()
	if err != nil {
		panic("could not set up metrics")
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

func (m mi) Gauge() (prometheus.Gauge, error) {
	if m.gauge == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.gauge, nil
}

func (m mi) GaugeVec() (*prometheus.GaugeVec, error) {
	if m.gaugeV == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.gaugeV, nil
}

func (m mi) Counter() (prometheus.Counter, error) {
	if m.counter == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.counter, nil
}

func (m mi) CounterVec() (*prometheus.CounterVec, error) {
	if m.counterV == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.counterV, nil
}

func (m mi) Histogram() (prometheus.Histogram, error) {
	if m.histogram == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.histogram, nil
}

func (m mi) HistogramVec() (*prometheus.HistogramVec, error) {
	if m.histogramV == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.histogramV, nil
}

func (m mi) Summary() (prometheus.Summary, error) {
	if m.summary == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.summary, nil
}

func (m mi) SummaryVec() (*prometheus.SummaryVec, error) {
	if m.summaryV == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.summaryV, nil
}

func setupMetrics() error {
	// instrument with time histogram for blocks
	h, err := AddInstrument(
		Counter,
		"engine_seconds_total",
		Namespace("vega"),
		Vectors("market", "engine", "fn"),
	)
	if err != nil {
		return err
	}
	est, err := h.CounterVec()
	if err != nil {
		return err
	}
	engineTime = est

	h, err = AddInstrument(
		Counter,
		"orders_total",
		Namespace("vega"),
		Vectors("market", "valid"),
		Help("Number of orders processed"),
	)
	if err != nil {
		return err
	}
	ot, err := h.CounterVec()
	if err != nil {
		return err
	}
	orderCounter = ot

	// now add the orders gauge
	h, err = AddInstrument(
		Gauge,
		"orders",
		Namespace("vega"),
		Vectors("market"),
		Help("Number of orders currently being processed"),
	)
	if err != nil {
		return err
	}
	g, err := h.GaugeVec()
	if err != nil {
		return err
	}
	orderGauge = g
	// example usage of this simple gauge:
	// e.orderGauge.WithLabelValues(mkt.Name).Add(float64(len(orders)))
	// e.orderGauge.WithLabelValues(mkt.Name).Sub(float64(len(completedOrders)))

	h, err = AddInstrument(
		Gauge,
		"unconfirmedtx",
		Namespace("vega"),
		Help("Number of transactions waiting to be processed"),
	)
	if err != nil {
		return err
	}
	utxg, err := h.Gauge()
	if err != nil {
		return err
	}
	unconfirmedTxGauge = utxg

	return nil
}

// EngineTimeCounterAdd is used to time a function. Call it, using defer, at the start of the
// function to be timed.
//
// e.g.
//     defer metrics.EngineTimeCounterAdd("x", "y", "z")()
//
// Note the extra "()" at the end of the above line - the returned function must be called.
func EngineTimeCounterAdd(labelValues ...string) func() {
	start := time.Now()
	f := func() {
		// Check that the metric has been set up. (Testing does not use metrics.)
		if engineTime == nil {
			return
		}
		engineTime.WithLabelValues(labelValues...).Add(time.Now().Sub(start).Seconds())
	}
	return f
}

func OrderCounterInc(labelValues ...string) {
	if orderCounter == nil {
		return
	}
	orderCounter.WithLabelValues(labelValues...).Inc()
}

func OrderGaugeAdd(n int, labelValues ...string) {
	if orderGauge == nil {
		return
	}
	orderGauge.WithLabelValues(labelValues...).Add(float64(n))
}

func UnconfirmedTxGaugeSet(n int) {
	if unconfirmedTxGauge == nil {
		return
	}
	unconfirmedTxGauge.Set(float64(n))
}

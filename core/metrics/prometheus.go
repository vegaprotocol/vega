// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package metrics

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/protos"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// Gauge ...
	Gauge instrument = iota
	// Counter ...
	Counter
	// Histogram ...
	Histogram
	// Summary ...
	Summary
)

var (
	// ErrInstrumentNotSupported signals the specified instrument is not yet supported.
	ErrInstrumentNotSupported = errors.New("instrument type unsupported")
	// ErrInstrumentTypeMismatch signal the type of the instrument is not expected.
	ErrInstrumentTypeMismatch = errors.New("instrument is not of the expected type")
)

var (
	unconfirmedTxGauge                      prometheus.Gauge
	engineTime                              *prometheus.CounterVec
	orderCounter                            *prometheus.CounterVec
	dataSourceEthVerifierOnGoingCallCounter *prometheus.CounterVec
	ethCallCounter                          *prometheus.CounterVec
	evtForwardCounter                       *prometheus.CounterVec
	orderGauge                              *prometheus.GaugeVec
	dataSourceEthVerifierOnGoingCallGauge   *prometheus.GaugeVec
	// Call counters for each request type per API.
	apiRequestCallCounter *prometheus.CounterVec
	// Total time counters for each request type per API.
	apiRequestTimeCounter *prometheus.CounterVec
	// Total time spent snapshoting.
	snapshotTimeGauge *prometheus.GaugeVec
	// Size of the snapshot per namespace.
	snapshotSizeGauge *prometheus.GaugeVec
	// Height of the last snapshot.
	snapshotBlockHeightCounter prometheus.Gauge
	// Core HTTP bindings that we will check against when updating HTTP metrics.
	httpBindings *protos.Bindings
)

// abstract prometheus types.
type instrument int

// combine all possible prometheus options + way to differentiate between regular or vector type.
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

// MetricInstrument - template interface for mi type return value - only mock if needed, and only mock the funcs you use.
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

// InstrumentOption - vararg for instrument options setting.
type InstrumentOption func(o *instrumentOpts)

// Vectors - configuration used to create a vector of a given interface, slice of label names.
func Vectors(labels ...string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.vectors = labels
	}
}

// Help - set the help field on instrument.
func Help(help string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.Help = help
	}
}

// Namespace - set namespace.
func Namespace(ns string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.Namespace = ns
	}
}

// Subsystem - set subsystem... obviously.
func Subsystem(s string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.Subsystem = s
	}
}

// Labels set labels for instrument (similar to vector, but with given values).
func Labels(labels map[string]string) InstrumentOption {
	return func(o *instrumentOpts) {
		o.opts.ConstLabels = labels
	}
}

// Buckets - specific to histogram type.
func Buckets(b []float64) InstrumentOption {
	return func(o *instrumentOpts) {
		o.buckets = b
	}
}

// Objectives - specific to summary type.
func Objectives(obj map[float64]float64) InstrumentOption {
	return func(o *instrumentOpts) {
		o.objectives = obj
	}
}

// MaxAge - specific to summary type.
func MaxAge(m time.Duration) InstrumentOption {
	return func(o *instrumentOpts) {
		o.maxAge = m
	}
}

// AgeBuckets - specific to summary type.
func AgeBuckets(ab uint32) InstrumentOption {
	return func(o *instrumentOpts) {
		o.ageBuckets = ab
	}
}

// BufCap - specific to summary type.
func BufCap(bc uint32) InstrumentOption {
	return func(o *instrumentOpts) {
		o.bufCap = bc
	}
}

// addInstrument configures and registers new metrics instrument.
// This will, over time, be moved to use custom Registries, etc...
func addInstrument(t instrument, name string, opts ...InstrumentOption) (*mi, error) {
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

// Start enable metrics (given config).
func Start(conf Config) {
	if !conf.Enabled {
		return
	}
	if err := setupMetrics(); err != nil {
		panic(fmt.Sprintf("could not set up metrics: %v", err))
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

// Gauge returns a prometheus Gauge instrument.
func (m mi) Gauge() (prometheus.Gauge, error) {
	if m.gauge == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.gauge, nil
}

// GaugeVec returns a prometheus GaugeVec instrument.
func (m mi) GaugeVec() (*prometheus.GaugeVec, error) {
	if m.gaugeV == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.gaugeV, nil
}

// Counter returns a prometheus Counter instrument.
func (m mi) Counter() (prometheus.Counter, error) {
	if m.counter == nil {
		return nil, ErrInstrumentTypeMismatch
	}
	return m.counter, nil
}

// CounterVec returns a prometheus CounterVec instrument.
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
	h, err := addInstrument(
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

	h, err = addInstrument(
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

	h, err = addInstrument(
		Counter,
		"data_source_ethverifier_calls_total",
		Namespace("vega"),
		Vectors("spec"),
		Help("Number of orders processed"),
	)
	if err != nil {
		return err
	}
	dataC, err := h.CounterVec()
	if err != nil {
		return err
	}
	dataSourceEthVerifierOnGoingCallCounter = dataC

	h, err = addInstrument(
		Counter,
		"eth_calls_total",
		Namespace("vega"),
		Vectors("func", "asset", "respcode"),
		Help("Number of call made to the ethereum node"),
	)
	if err != nil {
		return err
	}
	ethCalls, err := h.CounterVec()
	if err != nil {
		return err
	}
	ethCallCounter = ethCalls

	h, err = addInstrument(
		Counter,
		"evt_forward_total",
		Namespace("vega"),
		Vectors("func", "res"),
		Help("Number of call made forward/ack event from ethereum"),
	)
	if err != nil {
		return err
	}
	evtFwd, err := h.CounterVec()
	if err != nil {
		return err
	}
	evtForwardCounter = evtFwd

	// now add the orders gauge
	h, err = addInstrument(
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

	// now add the orders gauge
	h, err = addInstrument(
		Gauge,
		"data_source_ethverifier_calls_ongoing",
		Namespace("vega"),
		Vectors("spec"),
		Help("Number of event being verified"),
	)
	if err != nil {
		return err
	}
	dataD, err := h.GaugeVec()
	if err != nil {
		return err
	}
	dataSourceEthVerifierOnGoingCallGauge = dataD

	// example usage of this simple gauge:
	// e.orderGauge.WithLabelValues(mkt.Name).Add(float64(len(orders)))
	// e.orderGauge.WithLabelValues(mkt.Name).Sub(float64(len(completedOrders)))

	h, err = addInstrument(
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

	//
	// API usage metrics start here
	//

	httpBindings, err = protos.CoreBindings()
	if err != nil {
		return err
	}
	// Number of calls to each request type
	h, err = addInstrument(
		Counter,
		"request_count_total",
		Namespace("vega"),
		Vectors("apiType", "requestType"),
		Help("Count of API requests"),
	)
	if err != nil {
		return err
	}
	rc, err := h.CounterVec()
	if err != nil {
		return err
	}
	apiRequestCallCounter = rc

	// Total time for calls to each request type for each api type
	h, err = addInstrument(
		Counter,
		"request_time_total",
		Namespace("vega"),
		Vectors("apiType", "requestType"),
		Help("Total time spent in each API request"),
	)
	if err != nil {
		return err
	}
	rpac, err := h.CounterVec()
	if err != nil {
		return err
	}
	apiRequestTimeCounter = rpac

	// snapshots times
	h, err = addInstrument(
		Gauge,
		"snapshot_time_seconds",
		Namespace("vega"),
		Vectors("engine"),
		Help("Total time spent snapshotting state"),
	)
	if err != nil {
		return err
	}
	snap, err := h.GaugeVec()
	if err != nil {
		return err
	}
	snapshotTimeGauge = snap

	// snapshots sizes
	h, err = addInstrument(
		Gauge,
		"snapshot_size_bytes",
		Namespace("vega"),
		Vectors("engine"),
		Help("Total size of the snapshotting state"),
	)
	if err != nil {
		return err
	}
	snapSize, err := h.GaugeVec()
	if err != nil {
		return err
	}
	snapshotSizeGauge = snapSize

	// snapshots block heights
	h, err = addInstrument(
		Gauge,
		"snapshot_block_height",
		Namespace("vega"),
		Help("Block height of the last snapshot"),
	)
	if err != nil {
		return err
	}
	snapBlockHeight, err := h.Gauge()
	if err != nil {
		return err
	}
	snapshotBlockHeightCounter = snapBlockHeight

	return nil
}

// OrderCounterInc increments the order counter.
func OrderCounterInc(labelValues ...string) {
	if orderCounter == nil {
		return
	}
	orderCounter.WithLabelValues(labelValues...).Inc()
}

// DataSourceEthVerifierCallCounterInc increments the order counter.
func DataSourceEthVerifierCallCounterInc(labelValues ...string) {
	if dataSourceEthVerifierOnGoingCallCounter == nil {
		return
	}
	dataSourceEthVerifierOnGoingCallCounter.WithLabelValues(labelValues...).Inc()
}

// EthCallInc increments the eth call counter.
func EthCallInc(labelValues ...string) {
	if ethCallCounter == nil {
		return
	}
	ethCallCounter.WithLabelValues(labelValues...).Inc()
}

// EvtForwardInc increments the evt forward counter.
func EvtForwardInc(labelValues ...string) {
	if evtForwardCounter == nil {
		return
	}
	evtForwardCounter.WithLabelValues(labelValues...).Inc()
}

// OrderGaugeAdd increment the order gauge.
func OrderGaugeAdd(n int, labelValues ...string) {
	if orderGauge == nil {
		return
	}
	orderGauge.WithLabelValues(labelValues...).Add(float64(n))
}

// DataSourceEthVerifierCallGaugeAdd increments the eth verified calls.
func DataSourceEthVerifierCallGaugeAdd(n int, labelValues ...string) {
	if dataSourceEthVerifierOnGoingCallGauge == nil {
		return
	}
	dataSourceEthVerifierOnGoingCallGauge.WithLabelValues(labelValues...).Add(float64(n))
}

func DataSourceEthVerifierCallGaugeReset(labelValues ...string) {
	if dataSourceEthVerifierOnGoingCallGauge == nil {
		return
	}
	dataSourceEthVerifierOnGoingCallGauge.WithLabelValues(labelValues...).Set(0)
}

// UnconfirmedTxGaugeSet update the number of unconfirmed transactions.
func UnconfirmedTxGaugeSet(n int) {
	if unconfirmedTxGauge == nil {
		return
	}
	unconfirmedTxGauge.Set(float64(n))
}

// APIRequestAndTimeREST updates the metrics for REST API calls.
func APIRequestAndTimeREST(method, request string, time float64) {
	if apiRequestCallCounter == nil || apiRequestTimeCounter == nil || httpBindings == nil {
		return
	}

	const (
		invalid = "invalid route"
		prefix  = "/"
	)

	if !httpBindings.HasRoute(method, request) {
		apiRequestCallCounter.WithLabelValues("REST", invalid).Inc()
		apiRequestTimeCounter.WithLabelValues("REST", invalid).Add(time)
		return
	}

	uri := request

	// Remove the first slash if it has one
	if strings.Index(uri, prefix) == 0 {
		uri = uri[len(prefix):]
	}
	// Trim the URI down to something useful
	if strings.Count(uri, "/") >= 1 {
		uri = uri[:strings.Index(uri, "/")]
	}

	apiRequestCallCounter.WithLabelValues("REST", uri).Inc()
	apiRequestTimeCounter.WithLabelValues("REST", uri).Add(time)
}

// APIRequestAndTimeGRPC updates the metrics for GRPC API calls.
func APIRequestAndTimeGRPC(request string, startTime time.Time) {
	if apiRequestCallCounter == nil || apiRequestTimeCounter == nil {
		return
	}
	apiRequestCallCounter.WithLabelValues("GRPC", request).Inc()
	duration := time.Since(startTime).Seconds()
	apiRequestTimeCounter.WithLabelValues("GRPC", request).Add(duration)
}

// APIRequestAndTimeGraphQL updates the metrics for GraphQL API calls.
func APIRequestAndTimeGraphQL(request string, time float64) {
	if apiRequestCallCounter == nil || apiRequestTimeCounter == nil {
		return
	}
	apiRequestCallCounter.WithLabelValues("GraphQL", request).Inc()
	apiRequestTimeCounter.WithLabelValues("GraphQL", request).Add(time)
}

// StartAPIRequestAndTimeGRPC updates the metrics for GRPC API calls.
func StartAPIRequestAndTimeGRPC(request string) func() {
	startTime := time.Now()
	return func() {
		if apiRequestCallCounter == nil || apiRequestTimeCounter == nil {
			return
		}
		apiRequestCallCounter.WithLabelValues("GRPC", request).Inc()
		duration := time.Since(startTime).Seconds()
		apiRequestTimeCounter.WithLabelValues("GRPC", request).Add(duration)
	}
}

func RegisterSnapshotNamespaces(
	namespace string,
	timeTaken time.Duration,
	size int,
) {
	if snapshotTimeGauge == nil || snapshotSizeGauge == nil {
		return
	}
	snapshotTimeGauge.WithLabelValues(namespace).Set(timeTaken.Seconds())
	snapshotSizeGauge.WithLabelValues(namespace).Set(float64(size))
}

func RegisterSnapshotBlockHeight(
	blockHeight uint64,
) {
	if snapshotBlockHeightCounter == nil {
		return
	}
	snapshotBlockHeightCounter.Set(float64(blockHeight))
}

func StartSnapshot(namespace string) func() {
	startTime := time.Now()
	return func() {
		if snapshotTimeGauge == nil {
			return
		}
		duration := time.Since(startTime).Seconds()
		snapshotTimeGauge.WithLabelValues(namespace).Set(duration)
	}
}

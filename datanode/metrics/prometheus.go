// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package metrics

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/protos"

	"code.vegaprotocol.io/vega/core/events"
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
	engineTime        *prometheus.CounterVec
	eventHandlingTime *prometheus.CounterVec
	flushHandlingTime *prometheus.CounterVec
	eventCounter      *prometheus.CounterVec
	sqlQueryTime      *prometheus.CounterVec
	sqlQueryCounter   *prometheus.CounterVec
	blockCounter      prometheus.Counter
	blockHandlingTime prometheus.Counter
	blockHeight       prometheus.Gauge

	publishedEventsCounter         *prometheus.CounterVec
	eventBusPublishedEventsCounter *prometheus.CounterVec

	// Subscription gauge for each type.
	subscriptionGauge         *prometheus.GaugeVec
	eventBusSubscriptionGauge *prometheus.GaugeVec
	eventBusConnectionGauge   prometheus.Gauge

	// Call counters for each request type per API.
	apiRequestCallCounter *prometheus.CounterVec
	// Total time counters for each request type per API.
	apiRequestTimeCounter *prometheus.CounterVec

	lastSnapshotRowcount          prometheus.Gauge
	lastSnapshotCurrentStateBytes prometheus.Gauge
	lastSnapshotHistoryBytes      prometheus.Gauge
	lastSnapshotSeconds           prometheus.Gauge

	eventBufferWrittenCount prometheus.Counter
	eventBufferReadCount    prometheus.Counter

	networkHistoryIpfsStoreBytes prometheus.Gauge
	// Data Node HTTP bindings that we will check against when updating HTTP metrics.
	httpBindings *protos.Bindings

	// Per table segment creation time.
	networkHistoryCopiedRowsCounter *prometheus.CounterVec
	networkHistoryCopyTimeCounter   *prometheus.CounterVec

	batcherAddedEntities   *prometheus.CounterVec
	batcherFlushedEntities *prometheus.CounterVec
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

// addInstrument  configure and register new metrics instrument
// this will, over time, be moved to use custom Registries, etc...
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
		Namespace("datanode"),
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
		"flush_handling_seconds_total",
		Namespace("datanode"),
		Vectors("subscriber"),
	)
	if err != nil {
		return err
	}
	fht, err := h.CounterVec()
	if err != nil {
		return err
	}
	flushHandlingTime = fht

	// eventHandlingTime
	h, err = addInstrument(
		Counter,
		"event_handling_seconds_total",
		Namespace("datanode"),
		Vectors("type", "subscriber", "event"),
	)
	if err != nil {
		return err
	}
	eht, err := h.CounterVec()
	if err != nil {
		return err
	}
	eventHandlingTime = eht

	h, err = addInstrument(
		Counter,
		"published_event_count_total",
		Namespace("datanode"),
		Vectors("event"),
	)
	if err != nil {
		return err
	}
	sec, err := h.CounterVec()
	if err != nil {
		return err
	}
	publishedEventsCounter = sec

	h, err = addInstrument(
		Counter,
		"event_bus_published_event_count_total",
		Namespace("datanode"),
		Vectors("event"),
	)
	if err != nil {
		return err
	}
	sec, err = h.CounterVec()
	if err != nil {
		return err
	}
	eventBusPublishedEventsCounter = sec

	// eventCount
	h, err = addInstrument(
		Counter,
		"event_count_total",
		Namespace("datanode"),
		Vectors("event"),
	)
	if err != nil {
		return err
	}
	ec, err := h.CounterVec()
	if err != nil {
		return err
	}
	eventCounter = ec

	h, err = addInstrument(
		Counter,
		"network_history_copied_rows_total",
		Namespace("datanode"),
		Vectors("table"),
	)
	if err != nil {
		return err
	}
	cr, err := h.CounterVec()
	if err != nil {
		return err
	}
	networkHistoryCopiedRowsCounter = cr

	h, err = addInstrument(
		Counter,
		"network_history_copy_time_total",
		Namespace("datanode"),
		Vectors("table"),
	)
	if err != nil {
		return err
	}
	ct, err := h.CounterVec()
	if err != nil {
		return err
	}
	networkHistoryCopyTimeCounter = ct

	h, err = addInstrument(
		Counter,
		"batcher_added_entities",
		Namespace("datanode"),
		Vectors("table"),
	)
	if err != nil {
		return err
	}
	baet, err := h.CounterVec()
	if err != nil {
		return err
	}
	batcherAddedEntities = baet

	h, err = addInstrument(
		Counter,
		"batcher_flushed_entities",
		Namespace("datanode"),
		Vectors("table"),
	)
	if err != nil {
		return err
	}
	bfe, err := h.CounterVec()
	if err != nil {
		return err
	}
	batcherFlushedEntities = bfe

	// sqlQueryTime
	h, err = addInstrument(
		Counter,
		"sql_query_seconds_total",
		Namespace("datanode"),
		Vectors("store", "query"),
	)
	if err != nil {
		return err
	}
	sqt, err := h.CounterVec()
	if err != nil {
		return err
	}
	sqlQueryTime = sqt

	// sqlQueryCounter
	h, err = addInstrument(
		Counter,
		"sql_query_count",
		Namespace("datanode"),
		Vectors("store", "query"),
	)
	if err != nil {
		return err
	}
	qc, err := h.CounterVec()
	if err != nil {
		return err
	}
	sqlQueryCounter = qc

	h, err = addInstrument(
		Counter,
		"blocks_handling_time_seconds_total",
		Namespace("datanode"),
		Vectors(),
		Help("Total time handling blocks"),
	)
	if err != nil {
		return err
	}
	bht, err := h.Counter()
	if err != nil {
		return err
	}
	blockHandlingTime = bht

	h, err = addInstrument(
		Counter,
		"blocks_total",
		Namespace("datanode"),
		Vectors(),
		Help("Number of blocks processed"),
	)
	if err != nil {
		return err
	}
	bt, err := h.Counter()
	if err != nil {
		return err
	}
	blockCounter = bt

	h, err = addInstrument(
		Counter,
		"event_buffer_written_count",
		Namespace("datanode"),
		Vectors(),
		Help("Total number of event written to the event buffer"),
	)
	if err != nil {
		return err
	}
	ebwc, err := h.Counter()
	if err != nil {
		return err
	}
	eventBufferWrittenCount = ebwc

	h, err = addInstrument(
		Counter,
		"event_buffer_read_count",
		Namespace("datanode"),
		Vectors(),
		Help("Total number of events read from the event buffer"),
	)
	if err != nil {
		return err
	}
	ebrc, err := h.Counter()
	if err != nil {
		return err
	}
	eventBufferReadCount = ebrc

	h, err = addInstrument(
		Gauge,
		"block_height",
		Namespace("datanode"),
		Vectors(),
		Help("Current block height"),
	)
	if err != nil {
		return err
	}
	bh, err := h.Gauge()
	if err != nil {
		return err
	}
	blockHeight = bh

	h, err = addInstrument(
		Gauge,
		"last_snapshot_rowcount",
		Namespace("datanode"),
		Vectors(),
		Help("Last Snapshot Row Count"),
	)
	if err != nil {
		return err
	}
	lsr, err := h.Gauge()
	if err != nil {
		return err
	}
	lastSnapshotRowcount = lsr

	h, err = addInstrument(
		Gauge,
		"last_snapshot_history_bytes",
		Namespace("datanode"),
		Vectors(),
		Help("The compressed byte size of the last snapshots history data"),
	)
	if err != nil {
		return err
	}
	lshb, err := h.Gauge()
	if err != nil {
		return err
	}
	lastSnapshotHistoryBytes = lshb

	h, err = addInstrument(
		Gauge,
		"last_snapshot_current_state_bytes",
		Namespace("datanode"),
		Vectors(),
		Help("The compressed byte size of the last snapshots current state data"),
	)
	if err != nil {
		return err
	}
	lscs, err := h.Gauge()
	if err != nil {
		return err
	}
	lastSnapshotCurrentStateBytes = lscs

	h, err = addInstrument(
		Gauge,
		"last_snapshot_seconds",
		Namespace("datanode"),
		Vectors(),
		Help("Last Snapshot Time Taken in Seconds"),
	)
	if err != nil {
		return err
	}
	lss, err := h.Gauge()
	if err != nil {
		return err
	}
	lastSnapshotSeconds = lss

	h, err = addInstrument(
		Gauge,
		"networkhistory_ipfs_store_bytes",
		Namespace("datanode"),
		Vectors(),
		Help("The size in bytes of the network history ipfs store"),
	)
	if err != nil {
		return err
	}
	dsb, err := h.Gauge()
	if err != nil {
		return err
	}
	networkHistoryIpfsStoreBytes = dsb

	//
	// API usage metrics start here
	//

	if h, err = addInstrument(
		Gauge,
		"active_subscriptions",
		Namespace("datanode"),
		Vectors("apiType", "eventType"),
		Help("Number of active subscriptions"),
	); err != nil {
		return err
	}

	if subscriptionGauge, err = h.GaugeVec(); err != nil {
		return err
	}

	if h, err = addInstrument(
		Gauge,
		"event_bus_active_subscriptions",
		Namespace("datanode"),
		Vectors("eventType"),
		Help("Number of active subscriptions by type to the event bus"),
	); err != nil {
		return err
	}

	if eventBusSubscriptionGauge, err = h.GaugeVec(); err != nil {
		return err
	}

	if h, err = addInstrument(
		Gauge,
		"event_bus_active_connections",
		Namespace("datanode"),
		Help("Number of active connections to the event bus"),
	); err != nil {
		return err
	}
	ac, err := h.Gauge()
	if err != nil {
		return err
	}
	eventBusConnectionGauge = ac

	httpBindings, err = protos.DataNodeBindings()
	if err != nil {
		return err
	}
	// Number of calls to each request type
	h, err = addInstrument(
		Counter,
		"request_count_total",
		Namespace("datanode"),
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
		Namespace("datanode"),
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

	return nil
}

func AddBlockHandlingTime(duration time.Duration) {
	if blockHandlingTime != nil {
		blockHandlingTime.Add(duration.Seconds())
	}
}

func EventBufferWrittenCountInc() {
	if eventBufferWrittenCount == nil {
		return
	}
	eventBufferWrittenCount.Inc()
}

func EventBufferReadCountInc() {
	if eventBufferReadCount == nil {
		return
	}
	eventBufferReadCount.Inc()
}

func BlockCounterInc(labelValues ...string) {
	if blockCounter == nil {
		return
	}
	blockCounter.Inc()
}

func EventCounterInc(labelValues ...string) {
	if eventCounter == nil {
		return
	}
	eventCounter.WithLabelValues(labelValues...).Inc()
}

func PublishedEventsAdd(event string, eventCount float64) {
	if publishedEventsCounter == nil {
		return
	}

	publishedEventsCounter.WithLabelValues(event).Add(eventCount)
}

func EventBusPublishedEventsAdd(event string, eventCount float64) {
	if eventBusPublishedEventsCounter == nil {
		return
	}

	eventBusPublishedEventsCounter.WithLabelValues(event).Add(eventCount)
}

func SetBlockHeight(height float64) {
	if blockHeight == nil {
		return
	}
	blockHeight.Set(height)
}

func SetLastSnapshotRowcount(count float64) {
	if lastSnapshotRowcount == nil {
		return
	}
	lastSnapshotRowcount.Set(count)
}

func SetLastSnapshotCurrentStateBytes(bytes float64) {
	if lastSnapshotCurrentStateBytes == nil {
		return
	}
	lastSnapshotCurrentStateBytes.Set(bytes)
}

func SetLastSnapshotHistoryBytes(bytes float64) {
	if lastSnapshotHistoryBytes == nil {
		return
	}
	lastSnapshotHistoryBytes.Set(bytes)
}

func SetLastSnapshotSeconds(seconds float64) {
	if lastSnapshotSeconds == nil {
		return
	}
	lastSnapshotSeconds.Set(seconds)
}

func SetNetworkHistoryIpfsStoreBytes(bytes float64) {
	if networkHistoryIpfsStoreBytes == nil {
		return
	}
	networkHistoryIpfsStoreBytes.Set(bytes)
}

// APIRequestAndTimeREST updates the metrics for REST API calls.
func APIRequestAndTimeREST(method, request string, time float64) {
	if apiRequestCallCounter == nil || apiRequestTimeCounter == nil || httpBindings == nil {
		return
	}

	const (
		invalid = "invalid route"
		prefix  = "/api/v2/"
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

func IncrementBatcherAddedEntities(table string) {
	if batcherAddedEntities == nil {
		return
	}
	batcherAddedEntities.WithLabelValues(table).Add(1)
}

func BatcherFlushedEntitiesAdd(table string, flushed int) {
	if batcherFlushedEntities == nil {
		return
	}
	batcherFlushedEntities.WithLabelValues(table).Add(float64(flushed))
}

func NetworkHistoryRowsCopied(table string, rowsCopied int64) {
	if networkHistoryCopiedRowsCounter == nil {
		return
	}
	networkHistoryCopiedRowsCounter.WithLabelValues(table).Add(float64(rowsCopied))
}

func StartNetworkHistoryCopy(table string) func() {
	startTime := time.Now()
	return func() {
		if networkHistoryCopyTimeCounter == nil {
			return
		}
		duration := time.Since(startTime).Seconds()
		networkHistoryCopyTimeCounter.WithLabelValues(table).Add(duration)
	}
}

func StartSQLQuery(store string, query string) func() {
	startTime := time.Now()
	return func() {
		if sqlQueryTime == nil || sqlQueryCounter == nil {
			return
		}
		sqlQueryCounter.WithLabelValues(store, query).Inc()
		duration := time.Since(startTime).Seconds()
		sqlQueryTime.WithLabelValues(store, query).Add(duration)
	}
}

func StartActiveSubscriptionCountGRPC(subscribedToType string) func() {
	if subscriptionGauge == nil {
		return func() {}
	}

	subscriptionGauge.WithLabelValues("GRPC", subscribedToType).Inc()
	return func() {
		subscriptionGauge.WithLabelValues("GRPC", subscribedToType).Dec()
	}
}

func StartActiveEventBusConnection() func() {
	if eventBusConnectionGauge == nil {
		return func() {}
	}

	eventBusConnectionGauge.Inc()
	return func() {
		eventBusConnectionGauge.Dec()
	}
}

func StartEventBusActiveSubscriptionCount(eventTypes []events.Type) {
	if eventBusSubscriptionGauge == nil {
		return
	}

	for _, eventType := range eventTypes {
		eventBusSubscriptionGauge.WithLabelValues(eventType.String()).Inc()
	}
}

func StopEventBusActiveSubscriptionCount(eventTypes []events.Type) {
	if eventBusSubscriptionGauge == nil {
		return
	}

	for _, eventType := range eventTypes {
		eventBusSubscriptionGauge.WithLabelValues(eventType.String()).Dec()
	}
}

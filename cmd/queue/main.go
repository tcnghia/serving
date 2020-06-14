/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/types"

	"knative.dev/networking/pkg/apis/networking"
	pkglogging "knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/pkg/metrics"
	pkgnet "knative.dev/pkg/network"
	"knative.dev/pkg/profiling"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/tracing"
	tracingconfig "knative.dev/pkg/tracing/config"
	"knative.dev/serving/pkg/activator"
	activatorutil "knative.dev/serving/pkg/activator/util"
	pkghttp "knative.dev/serving/pkg/http"
	"knative.dev/serving/pkg/http/handler"
	"knative.dev/serving/pkg/logging"
	"knative.dev/serving/pkg/network"
	"knative.dev/serving/pkg/queue"
	"knative.dev/serving/pkg/queue/health"
	"knative.dev/serving/pkg/queue/readiness"
)

const (
	badProbeTemplate = "unexpected probe header value: %s"

	// reportingPeriod is the interval of time between reporting stats by queue proxy.
	reportingPeriod = 1 * time.Second
)

var (
	logger *zap.SugaredLogger

	readinessProbeTimeout = flag.Int("probe-period", -1, "run readiness probe with given timeout")
)

type config struct {
	ContainerConcurrency   int    `split_words:"true" required:"true"`
	QueueServingPort       int    `split_words:"true" required:"true"`
	UserPort               int    `split_words:"true" required:"true"`
	RevisionTimeoutSeconds int    `split_words:"true" required:"true"`
	ServingReadinessProbe  string `split_words:"true" required:"true"`
	EnableProfiling        bool   `split_words:"true"` // optional

	// Logging configuration
	ServingLoggingConfig         string `split_words:"true" required:"true"`
	ServingLoggingLevel          string `split_words:"true" required:"true"`
	ServingRequestLogTemplate    string `split_words:"true"` // optional
	ServingEnableProbeRequestLog bool   `split_words:"true"` // optional

	// Metrics configuration
	ServingNamespace             string `split_words:"true" required:"true"`
	ServingRevision              string `split_words:"true" required:"true"`
	ServingConfiguration         string `split_words:"true" required:"true"`
	ServingPodIP                 string `split_words:"true" required:"true"`
	ServingPod                   string `split_words:"true" required:"true"`
	ServingService               string `split_words:"true"` // optional
	ServingRequestMetricsBackend string `split_words:"true"` // optional

	// Tracing configuration
	TracingConfigDebug                bool                      `split_words:"true"` // optional
	TracingConfigBackend              tracingconfig.BackendType `split_words:"true"` // optional
	TracingConfigSampleRate           float64                   `split_words:"true"` // optional
	TracingConfigZipkinEndpoint       string                    `split_words:"true"` // optional
	TracingConfigStackdriverProjectID string                    `split_words:"true"` // optional
}

// TODO: This function does not belong here, probably should be in pkg.
//
// Provides an 'automatic downgrading' transport.
//
// (1) when health.State.Protocol == HTTP2, use the AutoTransport unmodified.
//
// (2) when health.State.Protocol isn't:
//     - a request will be downgraded to HTTP1 and send through the AutoTransport (thus taking the HTTP1 code path and wire formate).
//     - the response from that request will be then have the protocol version converted back to HTTP2 before sending back.
func autoDowngradingTransport(healthState *health.State) http.RoundTripper {
	t := pkgnet.AutoTransport
	return pkgnet.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {

		// Can the user container handle HTTP2 request?
		isHTTP2 := healthState.IsHTTP2()

		// If the user-container can handle HTTP2, we pass through the request as-is.
		if isHTTP2 != nil && *isHTTP2 {
			return t.RoundTrip(r)
		}

		// Otherwise, save the request HTTP version and downgrade it to HTTP1 before sending.
		version := r.ProtoMajor
		r.ProtoMajor = 1
		resp, err := t.RoundTrip(r)

		// Restore the request & response HTTP version before sending back.
		r.ProtoMajor = version
		if resp != nil {
			resp.ProtoMajor = version
		}
		return resp, err
	})
}

// Make handler a closure for testing.
func proxyHandler(breaker *queue.Breaker, stats *network.RequestStats, tracingEnabled bool, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if network.IsKubeletProbe(r) {
			next.ServeHTTP(w, r)
			return
		}

		if tracingEnabled {
			proxyCtx, proxySpan := trace.StartSpan(r.Context(), "queue_proxy")
			r = r.WithContext(proxyCtx)
			defer proxySpan.End()
		}

		// Metrics for autoscaling.
		in, out := network.ReqIn, network.ReqOut
		if activator.Name == network.KnativeProxyHeader(r) {
			in, out = network.ProxiedIn, network.ProxiedOut
		}
		stats.HandleEvent(network.ReqEvent{Time: time.Now(), Type: in})
		defer func() {
			stats.HandleEvent(network.ReqEvent{Time: time.Now(), Type: out})
		}()
		network.RewriteHostOut(r)

		// Enforce queuing and concurrency limits.
		if breaker != nil {
			var waitSpan *trace.Span
			if tracingEnabled {
				_, waitSpan = trace.StartSpan(r.Context(), "queue_wait")
			}
			if err := breaker.Maybe(r.Context(), func() {
				waitSpan.End()
				next.ServeHTTP(w, r)
			}); err != nil {
				waitSpan.End()
				switch err {
				case context.DeadlineExceeded, queue.ErrRequestQueueFull:
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
				default:
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		} else {
			next.ServeHTTP(w, r)
		}
	}
}

// Send an OPTIONS request to the user-container, with the Upgrade=h2c header.
//
// If we receive a HTTP 101 (SWITCHING PROTOCOLS),  the user-container _can_ support HTTP/2.
// If we receive any other HTTP response, the user-container can only support HTTP/1
func checkHTTP2() (*bool, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	url := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   "/",
	}
	req, err := http.NewRequest(http.MethodOptions, url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error constructing probe request %w", err)
	}

	// An upgrade will need to have at least these 3 headers.
	req.Header.Add("Connection", "Upgrade, HTTP2-Settings")
	req.Header.Add("Upgrade", "h2c")
	req.Header.Add("HTTP2-Settings", "")

	// This is not strictly required, but nice to have.
	req.Header.Add(network.UserAgentKey, network.KubeProbeUAPrefix)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	isHTTP2 := false

	// TODO: check the Upgrade response header to be more certain.
	if res.StatusCode == http.StatusSwitchingProtocols {
		isHTTP2 = true
	}
	return &isHTTP2, nil
}

func knativeProbeHandler(healthState *health.State, prober func() bool, isAggressive bool, tracingEnabled bool, next http.Handler, logger *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// We only check for HTTP/2 capacity once, if the capacity is confirmed or denied we won't check again.
		if healthState.IsHTTP2() == nil {
			isHTTP2, err := checkHTTP2()
			if err != nil || isHTTP2 == nil {
				http.Error(w, "fail to check protocol", http.StatusInternalServerError)
				return
			}
			if *isHTTP2 {
				logger.Infof("HTTP2 container detected. Happy!")
			} else {
				logger.Infof("HTTP1 container detected. WTF!")
			}
			healthState.SetHTTP2(*isHTTP2)
		}
		ph := network.KnativeProbeHeader(r)

		if ph == "" {
			next.ServeHTTP(w, r)
			return
		}

		var probeSpan *trace.Span
		if tracingEnabled {
			_, probeSpan = trace.StartSpan(r.Context(), "probe")
			defer probeSpan.End()
		}

		if ph != queue.Name {
			http.Error(w, fmt.Sprintf(badProbeTemplate, ph), http.StatusBadRequest)
			probeSpan.Annotate([]trace.Attribute{
				trace.StringAttribute("queueproxy.probe.error", fmt.Sprintf(badProbeTemplate, ph))}, "error")
			return
		}

		if prober == nil {
			http.Error(w, "no probe", http.StatusInternalServerError)
			probeSpan.Annotate([]trace.Attribute{
				trace.StringAttribute("queueproxy.probe.error", "no probe")}, "error")
			return
		}

		healthState.HandleHealthProbe(func() bool {
			if !prober() {
				probeSpan.Annotate([]trace.Attribute{
					trace.StringAttribute("queueproxy.probe.error", "container not ready")}, "error")
				return false
			}
			return true
		}, isAggressive, w)
	}
}

func main() {
	flag.Parse()

	// If this is set, we run as a standalone binary to probe the queue-proxy.
	if *readinessProbeTimeout >= 0 {
		os.Exit(standaloneProbeMain(time.Duration(*readinessProbeTimeout) * time.Second))
	}

	// Parse the environment.
	var env config
	if err := envconfig.Process("", &env); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Setup the logger.
	logger, _ = pkglogging.NewLogger(env.ServingLoggingConfig, env.ServingLoggingLevel)
	logger = logger.Named("queueproxy")
	defer flush(logger)

	logger = logger.With(
		zap.Object(logkey.Key, pkglogging.NamespacedName(types.NamespacedName{
			Namespace: env.ServingNamespace,
			Name:      env.ServingRevision,
		})),
		zap.String(logkey.Pod, env.ServingPod))

	// Report stats on Go memory usage every 30 seconds.
	msp := metrics.NewMemStatsAll()
	msp.Start(context.Background(), 30*time.Second)
	if err := view.Register(msp.DefaultViews()...); err != nil {
		logger.Fatalw("Error exporting go memstats view", zap.Error(err))
	}

	// Setup reporters and processes to handle stat reporting.
	promStatReporter, err := queue.NewPrometheusStatsReporter(
		env.ServingNamespace, env.ServingConfiguration, env.ServingRevision,
		env.ServingPod, reportingPeriod)
	if err != nil {
		logger.Fatalw("Failed to create stats reporter", zap.Error(err))
	}

	reportTicker := time.NewTicker(reportingPeriod)
	defer reportTicker.Stop()

	stats := network.NewRequestStats(time.Now())
	go func() {
		for now := range reportTicker.C {
			promStatReporter.Report(stats.Report(now))
		}
	}()

	// Setup probe to run for checking user-application healthiness.
	probe := buildProbe(env.ServingReadinessProbe)
	healthState := &health.State{}

	server := buildServer(env, healthState, probe, stats, logger)
	adminServer := buildAdminServer(healthState, logger)
	metricsServer := buildMetricsServer(promStatReporter)

	servers := map[string]*http.Server{
		"main":    server,
		"admin":   adminServer,
		"metrics": metricsServer,
	}

	if env.EnableProfiling {
		servers["profile"] = profiling.NewServer(profiling.NewHandler(logger, true))
	}

	errCh := make(chan error, len(servers))
	for name, server := range servers {
		go func(name string, s *http.Server) {
			// Don't forward ErrServerClosed as that indicates we're already shutting down.
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("%s server failed: %w", name, err)
			}
		}(name, server)
	}

	// Blocks until we actually receive a TERM signal or one of the servers
	// exit unexpectedly. We fold both signals together because we only want
	// to act on the first of those to reach here.
	select {
	case err := <-errCh:
		logger.Errorw("Failed to bring up queue-proxy, shutting down.", zap.Error(err))
		// This extra flush is needed because defers are not handled via os.Exit calls.
		flush(logger)
		os.Exit(1)
	case <-signals.SetupSignalHandler():
		logger.Info("Received TERM signal, attempting to gracefully shutdown servers.")
		healthState.Shutdown(func() {
			logger.Infof("Sleeping %v to allow K8s propagation of non-ready state", pkgnet.DefaultDrainTimeout)
			time.Sleep(pkgnet.DefaultDrainTimeout)

			// Calling server.Shutdown() allows pending requests to
			// complete, while no new work is accepted.
			logger.Info("Shutting down main server")
			if err := server.Shutdown(context.Background()); err != nil {
				logger.Errorw("Failed to shutdown proxy server", zap.Error(err))
			}
			// Removing the main server from the shutdown logic as we've already shut it down.
			delete(servers, "main")
		})

		for serverName, srv := range servers {
			logger.Info("Shutting down server: ", serverName)
			if err := srv.Shutdown(context.Background()); err != nil {
				logger.Errorw("Failed to shutdown server", zap.String("server", serverName), zap.Error(err))
			}
		}
		logger.Info("Shutdown complete, exiting...")
	}
}

func buildProbe(probeJSON string) *readiness.Probe {
	coreProbe, err := readiness.DecodeProbe(probeJSON)
	if err != nil {
		logger.Fatalw("Queue container failed to parse readiness probe", zap.Error(err))
	}
	return readiness.NewProbe(coreProbe)
}

func buildServer(env config, healthState *health.State, rp *readiness.Probe, stats *network.RequestStats,
	logger *zap.SugaredLogger) *http.Server {
	target := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("127.0.0.1", strconv.Itoa(env.UserPort)),
	}

	httpProxy := httputil.NewSingleHostReverseProxy(target)
	httpProxy.Transport = buildTransport(env, healthState, logger)
	httpProxy.ErrorHandler = pkgnet.ErrorHandler(logger)
	httpProxy.BufferPool = network.NewBufferPool()
	httpProxy.FlushInterval = -1
	activatorutil.SetupHeaderPruning(httpProxy)

	breaker := buildBreaker(env)
	metricsSupported := supportsMetrics(env, logger)
	tracingEnabled := env.TracingConfigBackend != tracingconfig.None
	timeout := time.Duration(env.RevisionTimeoutSeconds) * time.Second

	// Create queue handler chain.
	// Note: innermost handlers are specified first, ie. the last handler in the chain will be executed first.
	var composedHandler http.Handler = httpProxy
	if metricsSupported {
		composedHandler = requestAppMetricsHandler(composedHandler, breaker, env)
	}
	composedHandler = proxyHandler(breaker, stats, tracingEnabled, composedHandler)
	composedHandler = queue.ForwardedShimHandler(composedHandler)
	composedHandler = handler.NewTimeToFirstByteTimeoutHandler(composedHandler, "request timeout", func(*http.Request) time.Duration {
		return timeout
	})

	if metricsSupported {
		composedHandler = requestMetricsHandler(composedHandler, env)
	}
	composedHandler = tracing.HTTPSpanMiddleware(composedHandler)

	composedHandler = knativeProbeHandler(healthState, rp.ProbeContainer, rp.IsAggressive(), tracingEnabled, composedHandler, logger)
	composedHandler = network.NewProbeHandler(composedHandler)
	// We might want sometimes capture the probes/healthchecks in the request
	// logs. Hence we need to have RequestLogHandler to be the first one.
	composedHandler = pushRequestLogHandler(composedHandler, env)

	return pkgnet.NewServer(":"+strconv.Itoa(env.QueueServingPort), composedHandler)
}

func buildTransport(env config, healthState *health.State, logger *zap.SugaredLogger) http.RoundTripper {
	if env.TracingConfigBackend == tracingconfig.None {
		return autoDowngradingTransport(healthState)
	}

	oct := tracing.NewOpenCensusTracer(tracing.WithExporter(env.ServingPod, logger))
	oct.ApplyConfig(&tracingconfig.Config{
		Backend:              env.TracingConfigBackend,
		Debug:                env.TracingConfigDebug,
		ZipkinEndpoint:       env.TracingConfigZipkinEndpoint,
		StackdriverProjectID: env.TracingConfigStackdriverProjectID,
		SampleRate:           env.TracingConfigSampleRate,
	})

	return &ochttp.Transport{
		Base: autoDowngradingTransport(healthState),
	}
}

func buildBreaker(env config) *queue.Breaker {
	if env.ContainerConcurrency < 1 {
		return nil
	}

	// We set the queue depth to be equal to the container concurrency * 10 to
	// allow the autoscaler time to react.
	queueDepth := env.ContainerConcurrency * 10
	params := queue.BreakerParams{QueueDepth: queueDepth, MaxConcurrency: env.ContainerConcurrency, InitialCapacity: env.ContainerConcurrency}
	logger.Infof("Queue container is starting with %#v", params)

	return queue.NewBreaker(params)
}

func supportsMetrics(env config, logger *zap.SugaredLogger) bool {
	// Setup request metrics reporting for end-user metrics.
	if env.ServingRequestMetricsBackend == "" {
		return false
	}

	if err := setupMetricsExporter(env.ServingRequestMetricsBackend); err != nil {
		logger.Errorw("Error setting up request metrics exporter. Request metrics will be unavailable.", zap.Error(err))
		return false
	}

	return true
}

func buildAdminServer(healthState *health.State, logger *zap.SugaredLogger) *http.Server {
	adminMux := http.NewServeMux()
	drainHandler := healthState.DrainHandlerFunc()
	adminMux.HandleFunc(queue.RequestQueueDrainPath, func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Attached drain handler from user-container")
		drainHandler(w, r)
	})

	return &http.Server{
		Addr:    ":" + strconv.Itoa(networking.QueueAdminPort),
		Handler: adminMux,
	}
}

func buildMetricsServer(promStatReporter *queue.PrometheusStatsReporter) *http.Server {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promStatReporter.Handler())
	return &http.Server{
		Addr:    ":" + strconv.Itoa(networking.AutoscalingQueueMetricsPort),
		Handler: metricsMux,
	}
}

func pushRequestLogHandler(currentHandler http.Handler, env config) http.Handler {
	if env.ServingRequestLogTemplate == "" {
		return currentHandler
	}

	revInfo := &pkghttp.RequestLogRevision{
		Name:          env.ServingRevision,
		Namespace:     env.ServingNamespace,
		Service:       env.ServingService,
		Configuration: env.ServingConfiguration,
		PodName:       env.ServingPod,
		PodIP:         env.ServingPodIP,
	}
	handler, err := pkghttp.NewRequestLogHandler(currentHandler, logging.NewSyncFileWriter(os.Stdout), env.ServingRequestLogTemplate,
		pkghttp.RequestLogTemplateInputGetterFromRevision(revInfo), env.ServingEnableProbeRequestLog)
	if err != nil {
		logger.Errorw("Error setting up request logger. Request logs will be unavailable.", zap.Error(err))
		return currentHandler
	}
	return handler
}

func requestMetricsHandler(currentHandler http.Handler, env config) http.Handler {
	h, err := queue.NewRequestMetricsHandler(currentHandler, env.ServingNamespace,
		env.ServingService, env.ServingConfiguration, env.ServingRevision, env.ServingPod)
	if err != nil {
		logger.Errorw("Error setting up request metrics reporter. Request metrics will be unavailable.", zap.Error(err))
		return currentHandler
	}
	return h
}

func requestAppMetricsHandler(currentHandler http.Handler, breaker *queue.Breaker, env config) http.Handler {
	h, err := queue.NewAppRequestMetricsHandler(currentHandler, breaker, env.ServingNamespace,
		env.ServingService, env.ServingConfiguration, env.ServingRevision, env.ServingPod)
	if err != nil {
		logger.Errorw("Error setting up app request metrics reporter. Request metrics will be unavailable.", zap.Error(err))
		return currentHandler
	}
	return h
}

func setupMetricsExporter(backend string) error {
	// Set up OpenCensus exporter.
	// NOTE: We use revision as the component instead of queue because queue is
	// implementation specific. The current metrics are request relative. Using
	// revision is reasonable.
	// TODO(yanweiguo): add the ability to emit metrics with names not combined
	// to component.
	ops := metrics.ExporterOptions{
		Domain:         metrics.Domain(),
		Component:      "revision",
		PrometheusPort: networking.UserQueueMetricsPort,
		ConfigMap: map[string]string{
			metrics.BackendDestinationKey: backend,
		},
	}
	return metrics.UpdateExporter(ops, logger)
}

func flush(logger *zap.SugaredLogger) {
	logger.Sync()
	os.Stdout.Sync()
	os.Stderr.Sync()
	metrics.FlushExporter()
}

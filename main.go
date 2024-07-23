package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/retzkek/myjob/pkg/lens"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	address     = flag.String("a", "localhost:8888", "Address and port to listen on")
	serviceName = semconv.ServiceNameKey.String("myjob")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// The service name used to display traces in backends
			serviceName,
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	shutdownTracerProvider, err := initTracerProvider(ctx, res)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdownTracerProvider(ctx); err != nil {
			log.Fatalf("failed to shutdown TracerProvider: %s", err)
		}
	}()

	name := "github.com/retzkek/myjob"
	tracer := otel.Tracer(name)

	statusHandler := JobStatus{}
	http.Handle("/status/{jobid}", loggingHandler(statusHandler, tracer))

	http.Handle("/metrics", loggingHandler(promhttp.Handler(), tracer))

	fmt.Println("Listening on", *address)
	http.ListenAndServe(*address, nil)
}

type JobStatus struct{}

func (s JobStatus) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.getStatus(r.Context(), r.PathValue("jobid"), w); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s JobStatus) getStatus(ctx context.Context, jobid string, w io.Writer) error {
	j, err := lens.GetJobInfo(ctx, jobid)
	if err != nil {
		return err
	}

	done := "not done"
	if j.Done {
		done = "done"
	}
	fmt.Fprintf(w, "Subission %s submitted by %s at %s is %s.\n", jobid, j.Owner, j.SubmitTime.String(), done)
	return nil
}

var (
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of http request durations.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100, 250},
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(requestDuration)
}

// Initializes an OTLP exporter, and configures the corresponding trace provider.
func initTracerProvider(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	// Set up a trace exporter
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

// loggingHandler wraps an http.Handler to log each request
func loggingHandler(h http.Handler, t trace.Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.EscapedPath()
		var mpath string
		switch {
		case strings.HasPrefix(path, "/status"):
			mpath = "/status"
		case strings.HasPrefix(path, "/metrics"):
			mpath = "/metrics"
		default:
			mpath = "other"
		}
		// start tracing span
		// TODO extract from headers if possible
		ctx, span := t.Start(r.Context(), mpath)
		defer span.End()
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)

		// log completed request
		d := time.Since(start)
		log.WithFields(log.Fields{
			"origin":      originAddr(r),
			"length":      r.ContentLength,
			"path":        mpath,
			"method":      r.Method,
			"duration_ns": d.Nanoseconds(),
			"duration":    d.String(),
			"traceid":     span.SpanContext().TraceID().String(),
		}).Info("handled request")
		requestDuration.WithLabelValues(mpath).Observe(d.Seconds())
	})
}

// originAddr returns the "real" remote address for forwarded requests
func originAddr(r *http.Request) string {
	if remote := r.Header.Get("X-Real-IP"); remote != "" {
		return remote
	} else if remote := r.Header.Get("X-Forwarded-For"); remote != "" {
		return remote
	}
	return r.RemoteAddr
}

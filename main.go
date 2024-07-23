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
)

var (
	address = flag.String("a", "localhost:8888", "Address and port to listen on")
)

func main() {
	flag.Parse()

	statusHandler := JobStatus{}
	http.Handle("/status/{jobid}", loggingHandler(statusHandler))

	http.Handle("/metrics", loggingHandler(promhttp.Handler()))

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

// loggingHandler wraps an http.Handler to log each request
func loggingHandler(h http.Handler) http.Handler {
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

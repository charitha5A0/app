package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/server"
	"github.com/weaveworks/common/tracing"
)

func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func main() {
	serverConfig := server.Config{
		MetricsNamespace: "tns",
	}
	serverConfig.RegisterFlags(flag.CommandLine)
	flag.Parse()

	// Use a gokit logger, and tell the server to use it.
	logger := level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)), serverConfig.LogLevel.Gokit)
	serverConfig.Log = logging.GoKit(logger)

	// Setting the environment variable JAEGER_AGENT_HOST enables tracing
	trace, err := tracing.NewFromEnv("app")
	if err != nil {
		level.Error(logger).Log("msg", "error initializing tracing", "err", err)
		os.Exit(1)
	}
	defer trace.Close()

	s, err := server.New(serverConfig)
	if err != nil {
		level.Error(logger).Log("msg", "error starting server", "err", err)
		os.Exit(1)
	}
	defer s.Shutdown()
	app, err := new(logger)
	if err != nil {
		level.Error(logger).Log("msg", "error initialising app", "err", err)
		os.Exit(1)
	}
	http.HandleFunc("/hello", app.hello)
	http.HandleFunc("/post", app.post)
	http.HandleFunc("/web", app.web)
	recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8085", nil)
}

type app struct {
	logger log.Logger
	id     string
}

func new(logger log.Logger) (*app, error) {
	rand.Seed(time.Now().UnixNano())
	h := md5.New()
	fmt.Fprintf(h, "%d", rand.Int63())
	id := fmt.Sprintf("app-%x", h.Sum(nil))

	return &app{
		logger: logger,
		id:     id,
	}, nil
}
func (a *app) hello(w http.ResponseWriter, r *http.Request) {
	traceId, _ := tracing.ExtractTraceID(r.Context())
	fmt.Print(traceId)
	level.Info(a.logger).Log("msg", "hello page", "traceID", traceId)
	fmt.Fprintf(w, "Hello this web application hello page")
	w.WriteHeader(http.StatusOK)
}
func (a *app) post(w http.ResponseWriter, r *http.Request) {
	traceId, _ := tracing.ExtractTraceID(r.Context())
	level.Info(a.logger).Log("msg", "post page", "traceID", traceId)
	fmt.Fprintf(w, "hey this is post page")
	w.WriteHeader(http.StatusOK)
}
func (a *app) web(w http.ResponseWriter, r *http.Request) {
	traceId, _ := tracing.ExtractTraceID(r.Context())
	level.Info(a.logger).Log("msg", "Web page", "traceID", traceId)
	fmt.Fprintf(w, "Hello this is web page")
	w.WriteHeader(http.StatusOK)
}

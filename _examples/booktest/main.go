// Code generated by sqlc-grpc (https://github.com/walterwanderley/sqlc-grpc).

package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/flowchartsman/swaggerui"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// database driver
	_ "github.com/jackc/pgx/v4/stdlib"

	"booktest/internal/server"
	"booktest/internal/server/trace"
)

//go:generate sqlc-grpc -m booktest -append

const serviceName = "booktest"

var (
	dbURL string

	//go:embed api/apidocs.swagger.json
	openAPISpec []byte
)

func main() {
	cfg := server.Config{
		ServiceName: serviceName,
	}
	var dev bool
	flag.StringVar(&dbURL, "db", "", "The Database connection URL")
	flag.IntVar(&cfg.Port, "port", 5000, "The server port")
	flag.IntVar(&cfg.PrometheusPort, "prometheusPort", 0, "The metrics server port")
	flag.StringVar(&cfg.JaegerCollector, "jaegerCollector", "", "The Jaeger Tracing Collector endpoint (example: http://localhost:14268/api/traces)")
	flag.BoolVar(&cfg.EnableCors, "cors", false, "Enable CORS middleware")
	flag.BoolVar(&dev, "dev", false, "Set logger to development mode")

	flag.Parse()

	log := logger(dev)
	defer log.Sync()

	if err := run(cfg, log); err != nil && err.Error() != "mux: server closed" {
		log.Error("server error", zap.Error(err))
		os.Exit(1)
	}
}

func run(cfg server.Config, log *zap.Logger) error {
	if _, err := maxprocs.Set(); err != nil {
		log.Warn("startup", zap.Error(err))
	}
	log.Info("startup", zap.Int("GOMAXPROCS", runtime.GOMAXPROCS(0)))

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if cfg.TracingEnabled() {
		flush, err := trace.InitTracer(context.Background(), serviceName, cfg.JaegerCollector)
		if err != nil {
			return err
		}
		defer flush()

		db, err = trace.OpenDB(db.Driver(), dbURL)
		if err != nil {
			return err
		}
	}

	srv := server.New(cfg, log, registerServer(log, db), registerHandlers(), httpHandlers)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-done
		log.Warn("signal detected...", zap.Any("signal", sig))
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	return srv.ListenAndServe()
}

func logger(dev bool) *zap.Logger {
	var config zap.Config
	if dev {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
	}

	log, err := config.Build()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return log
}

func httpHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	mux.Handle("/swagger/", http.StripPrefix("/swagger", swaggerui.Handler(openAPISpec)))

}

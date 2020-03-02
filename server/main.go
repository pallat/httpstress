package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporter/trace/stdout"
	"go.opentelemetry.io/otel/plugin/othttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	buildcommit = "development"
	buildtime   = time.Now().Format(time.RFC3339)
)

func init() {
	initConfig()
	// log.Init()
	initTrace()
}

func main() {
	// defer log.Sync()

	r := mux.NewRouter()
	r.Use(mux.CORSMethodMiddleware(r))
	// r.Use(latencyMiddleware())

	r.HandleFunc("/build", buildHandler).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodOptions)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	srv := &http.Server{
		Handler: othttp.NewHandler(r, hostname,
			othttp.WithMessageEvents(othttp.ReadEvents, othttp.WriteEvents),
		),
		Addr:         "0.0.0.0:" + viper.GetString("port"),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Printf("serve on %s\n", ":"+viper.GetString("port"))
		log.Fatal(srv.ListenAndServe())
	}()

	shutdown(srv)
}

func buildHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")

	resp := map[string]string{
		"timestamp": buildtime,
		"commit":    buildcommit,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&resp)
}

func shutdown(srv *http.Server) {
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	<-sigterm

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Println(err.Error())
	}
}

func initTrace() {
	exporter, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(fmt.Errorf("have some errors while creating stdout exporter: %v", err))
	}

	provider, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		log.Fatal(fmt.Errorf("have some problems while creating provider: %v", err))
	}
	global.SetTraceProvider(provider)
}

func initConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Fatal error config file : %s \n", err))
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

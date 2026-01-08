package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BatuhanAri/rt-gateway/internal/metrics"
	"github.com/BatuhanAri/rt-gateway/internal/netws"
)

func main() {
	addr := envOr("RTG_ADDR", ":8083")

	m := metrics.New()
	ws := netws.NewServer(netws.Config{
		ReadLimitBytes:  64 * 1024, // M1: sane default
		PingInterval:    25 * time.Second,
		PongWait:        60 * time.Second,
		WriteTimeout:    5 * time.Second,
		CloseGrace:      2 * time.Second,
		MaxMessageBytes: 64 * 1024,
	}, m)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.Handle("/metrics", m.Handler())
	mux.HandleFunc("/ws", ws.HandleWS)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("shutting down...")
	_ = srv.Shutdown(ctx)
	log.Printf("bye")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

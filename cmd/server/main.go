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
	// Konfigürasyon: Port ortam değişkeninden alınır, yoksa varsayılan :8083 kullanılır.
	addr := envOr("RTG_ADDR", ":8083")

	m := metrics.New()
	// WebSocket Sunucu Ayarları (Güvenlik ve Performans limitleri)
	ws := netws.NewServer(netws.Config{
		ReadLimitBytes:  64 * 1024,        // Maksimum mesaj boyutu (64KB) - Bellek şişmesini önler.
		PingInterval:    25 * time.Second, // Bağlantıyı canlı tutmak için ping sıklığı.
		PongWait:        60 * time.Second, // Yanıt gelmezse bağlantıyı kesme süresi.
		WriteTimeout:    5 * time.Second,  // Yazma işlemi için zaman aşımı.
		CloseGrace:      2 * time.Second,  // Kapanış sırasında beklenecek süre.
		MaxMessageBytes: 64 * 1024,
	}, m)

	// -------------------------------------------------------------------------
	//  Router (Multiplexer) ve Endpoint Tanımları
	// -------------------------------------------------------------------------
	mux := http.NewServeMux()

	// [Health Check]
	// Servisin ayakta olduğunu ve istek kabul edebildiğini doğrular.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// 200 OK döner, karmaşık mantık içermez (hızlı yanıt için).
		_, _ = w.Write([]byte("ok"))
	})

	// [Observability]
	// (Goroutine sayısı, WS bağlantı sayısı, bellek vb.) çekebilmesi için metrikleri sunar.
	mux.Handle("/metrics", m.Handler())

	// [WebSocket Endpoint]
	// HTTP'den WebSocket protokolüne upgrade edilir.
	mux.HandleFunc("/ws", ws.HandleWS)

	// -------------------------------------------------------------------------
	//  HTTP Sunucu Konfigürasyonu (Security Hardening)
	// -------------------------------------------------------------------------
	srv := &http.Server{
		Addr:    addr, // Örn: :8083
		Handler: mux,  // router

		// [Güvenlik Kritik] Slowloris saldırılarına karşı koruma.
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Sunucuyu Başlat (Non-blocking)
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

	// Mevcut işlemlerin tamamlanması için 5 saniye süre tanır.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("shutting down...")
	_ = srv.Shutdown(ctx)
	log.Printf("bye")
}

// envOr ortam değişkenini okur, boşsa varsayılan değeri döndürür.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

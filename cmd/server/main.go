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

	//paketler
	"github.com/BatuhanAri/rt-gateway/internal/metrics"
	"github.com/BatuhanAri/rt-gateway/internal/netws"
)

func main() {
	// Hangi portta
	portAddress := envOr("RTG_ADDR", ":8083")

	// Metrikleri toplayacak olan objeyi oluşturuyorum.
	// New()
	metricSystem := metrics.New()

	// WebSocket ayarları
	// sunucu çökmesin diye limit koyuyoruz.
	wsConfig := netws.Config{
		ReadLimitBytes:  64 * 1024,        // 64 Kilobyte sınır koydum. Bundan büyük mesajları kabul etme.
		PingInterval:    25 * time.Second, // Her 25 saniyede bir "orada mısın?" diye kontrol et (Ping).
		PongWait:        60 * time.Second, // Eğer 60 saniye cevap gelmezse bağlantıyı kopar.
		WriteTimeout:    5 * time.Second,  // Mesaj gönderirken en fazla 5 saniye bekle, takılmasın.
		CloseGrace:      2 * time.Second,  // Kapatırken de hemen kesme, 2 saniye bekle.
		MaxMessageBytes: 64 * 1024,        // Bu da maksimum mesaj boyutu yine.
	}

	// WebSocket sunucusunu yukarıdaki ayarlarla ve metrik sistemiyle başlatıyorum.
	webSocketServer := netws.NewServer(wsConfig, metricSystem)

	// Router (Yönlendirici)
	// Gelen istekleri doğru yere yönlendiriyor.
	myRouter := http.NewServeMux()

	// -- Endpointler (Adresler) --

	// "/healthz" adresine istek gelince burası çalışacak.
	// Bu sadece sunucunun çalışıp çalışmadığını kontrol etmek için basit bir kontrol.
	myRouter.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Ekrana sadece "ok" yazıp bitiriyoruz. Hata kontrolü bile yapmadım hızlı olsun diye.
		w.Write([]byte("ok"))
	})

	// "/metrics" adresine gidince sistemin durumunu görebileceğiz.
	// Goroutine sayısı, RAM kullanımı gibi bilgileri veriyor.
	myRouter.Handle("/metrics", metricSystem.Handler())

	// "/ws" adresi asıl işi yapan yer. WebSocket bağlantısı burada kuruluyor.
	// http isteğini websocket'e çeviriyor (Upgrade ediyor).
	myRouter.HandleFunc("/ws", webSocketServer.HandleWS)

	// Sunucu ayarları
	// http.Server struct
	server := &http.Server{
		Addr:    portAddress, // Yukarıda belirlediğim port (8083)
		Handler: myRouter,    // Hangi yönlendiriciyi kullanacak?

		// "Slowloris" saldırısını engelliyo
		// Header okumak için en fazla 5 saniye bekliyor.
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Sunucuyu başlatıyorum ama "go" ile başlatıyorum.
	// Neden? Çünkü sunucu çalışırken aşağıdaki kodların da çalışmasını istiyorum (Shutdown için).
	// Eğer "go" koymazsam program burada takılı kalır.
	go func() {
		log.Printf("Sunucu şu adreste dinlemeye başlıyor: %s", portAddress)

		// ListenAndServe sunucuyu başlatır.
		err := server.ListenAndServe()

		// Eğer bir hata varsa ve bu hata "Sunucu Kapandı" hatası değilse ekrana basıp çık.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Sunucu başlatılamadı hata var: %v", err)
		}
	}()

	// Graceful Shutdown (Kibarca Kapatma)
	// Programı CTRL+C ile kapatınca lap diye kapanmasın, işleri bitirsin diye bunu yapıyoruz.

	// Bir kanal (channel) oluşturuyorum, sinyalleri dinleyecek.
	stopChannel := make(chan os.Signal, 1)

	// İşletim sistemine diyorum ki: "Biri programı durdurmaya çalışırsa (SIGINT, SIGTERM) bana haber ver".
	signal.Notify(stopChannel, syscall.SIGINT, syscall.SIGTERM)

	// Burada program bekliyor... Kanal'dan bir sinyal gelene kadar alt satıra geçmez.
	<-stopChannel

	// Sinyal geldi! Kapanma işlemi başlıyor.
	log.Printf("Kapatma sinyali alındı, sunucu kapatılıyor...")

	// Kapanırken maksimum 5 saniye beklesin diye zaman aşımı (timeout) oluşturuyorum.
	// context.Background() boş bir context demek.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Fonksiyon bitince cancel çalışsın diye defer koydum (hafıza temizliği).
	defer cancel()

	// Shutdown fonksiyonu ile sunucuyu nazikçe kapatıyorum.
	// Şu an açık olan bağlantıların bitmesini bekliyor (ama en fazla 5 sn).
	_ = server.Shutdown(ctx)

	log.Printf("Program bitti.")
}

// YARDIMCI FONKSİYON
// Ortam değişkeni
// Eğer key (anahtar) varsa onu döndürür, yoksa default (varsayılan) değeri döndürür.
func envOr(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val // Değer varsa onu döndür
	}
	return defaultValue // Yoksa varsayılanı döndür
}

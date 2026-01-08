package netws

import (
	"context"
	"net/http"
	"time"

	"github.com/BatuhanAri/rt-gateway/internal/metrics"
	"github.com/coder/websocket"
)

type Config struct {
	ReadLimitBytes  int64
	MaxMessageBytes int64

	PingInterval time.Duration
	PongWait     time.Duration
	WriteTimeout time.Duration
	CloseGrace   time.Duration
}

type Server struct {
	cfg Config
	m   *metrics.Metrics
}

func NewServer(cfg Config, m *metrics.Metrics) *Server {
	return &Server{cfg: cfg, m: m}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// M1: local dev. Production'da Origin check yapacaksın.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	// En kritik güvenlik/sağlamlık ayarı: read limit
	if s.cfg.ReadLimitBytes > 0 {
		c.SetReadLimit(s.cfg.ReadLimitBytes)
	}

	s.m.ConnectionsAccepted.Inc()
	s.m.ConnectionsCurrent.Inc()
	defer func() {
		s.m.ConnectionsCurrent.Dec()
		s.m.Disconnects.Inc()
		_ = c.Close(websocket.StatusNormalClosure, "bye")
	}()

	ctx := r.Context()

	for {
		msgType, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		s.m.MessagesIn.Inc()

		// Echo back (M1)
		writeCtx, cancel := context.WithTimeout(ctx, s.cfg.WriteTimeout)
		err = c.Write(writeCtx, msgType, data)
		cancel()
		if err != nil {
			return
		}
		s.m.MessagesOut.Inc()
	}
}

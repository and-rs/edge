package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	diagnosticsv1connect "github.com/index/edge/backend/gen/api/diagnostics/v1/diagnosticsv1connect"
	signalsv1connect "github.com/index/edge/backend/gen/api/signals/v1/signalsv1connect"
	"github.com/index/edge/backend/internal/app/signals"
	"github.com/index/edge/backend/internal/config"
)

type Server struct {
	cfg        config.Config
	httpServer *http.Server
}

func NewServer(cfg config.Config) (*Server, error) {
	mux := http.NewServeMux()

	diagnosticsService := NewDiagnosticsService()
	diagnosticsPath, diagnosticsHandler := diagnosticsv1connect.NewDiagnosticsServiceHandler(diagnosticsService)
	mux.Handle(diagnosticsPath, withCORS(diagnosticsHandler))

	judge, judgeErr := signals.NewSignalJudge(cfg.AI)
	if judgeErr != nil {
		log.Printf("ai judge unavailable: %v", judgeErr)
	}
	signalService := signals.NewService(cfg.AI, judge, judgeErr, cfg.SignalsCacheTTL)
	signalsPath, signalsHandler := signalsv1connect.NewSignalsServiceHandler(signalService)
	mux.Handle(signalsPath, withCORS(signalsHandler))

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           withRequestLogging(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		cfg:        cfg,
		httpServer: httpServer,
	}, nil
}

func (s *Server) Run() error {
	fmt.Printf("edge backend listening on %s\n", s.cfg.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web, X-User-Agent",
			)
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			w.Header().Set("Access-Control-Max-Age", "7200")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("http %s %s status=%d duration=%s", r.Method, r.URL.Path, recorder.status, time.Since(startedAt).Round(time.Millisecond))
	})
}

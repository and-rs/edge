package app

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/index/stint/backend/internal/config"
)

type Server struct {
    cfg        config.Config
    httpServer *http.Server
}

func NewServer(cfg config.Config) *Server {
    var mux *http.ServeMux = http.NewServeMux()
    mux.HandleFunc("/healthz", handleHealth)

    var httpServer *http.Server = &http.Server{
        Addr:              cfg.Addr,
        Handler:           mux,
        ReadHeaderTimeout: 5 * time.Second,
    }

    return &Server{
        cfg:        cfg,
        httpServer: httpServer,
    }
}

func (s *Server) Run() error {
    fmt.Printf("stint backend listening on %s\n", s.cfg.Addr)
    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("content-type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("ok\n"))
}

package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

type Server struct {
	port int
}

func NewServer(port int) *Server {
	return &Server{port: port}
}

func (hs *Server) handle(w http.ResponseWriter, r *http.Request) {
	select {
	case <-r.Context().Done():
		w.WriteHeader(http.StatusServiceUnavailable)
		break
	default:
		w.WriteHeader(http.StatusOK)
		break
	}
}

func (hs *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health-check", hs.handle)

	baseContextFunc := func(_ net.Listener) context.Context {
		return ctx
	}

	httpServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", hs.port),
		BaseContext: baseContextFunc,
		Handler:     mux,
	}

	return httpServer.ListenAndServe()
}

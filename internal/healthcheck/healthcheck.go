package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
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

	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", hs.port),
		BaseContext: baseContextFunc,
		Handler:     mux,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	<-ctx.Done()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		return fmt.Errorf("server shutdown failed:%v", err)
	}

	return nil
}

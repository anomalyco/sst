package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/sst/sst/v3/pkg/global"
	"github.com/sst/sst/v3/pkg/project"
	"github.com/sst/sst/v3/pkg/server/aws"
	"github.com/sst/sst/v3/pkg/server/resource"
	"github.com/sst/sst/v3/pkg/server/runtime"
	"github.com/sst/sst/v3/pkg/server/scrap"
)

type Server struct {
	Port int
	Mux  *http.ServeMux
	Rpc  *rpc.Server
}

func New() (*Server, error) {
	port, err := port()
	log := slog.Default().With("service", "server")
	log.Info("port assigned", "port", port)
	if err != nil {
		return nil, err
	}
	result := &Server{
		Port: port,
		Mux:  http.NewServeMux(),
		Rpc:  rpc.NewServer(),
	}

	result.Mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		log.Info("rpc request", "method", r.Method, "url", r.URL.String())
		result.Rpc.ServeCodec(jsonrpc.NewServerCodec(&HttpConn{Reader: r.Body, Writer: w}))
	})

	type JSONRPCRequest struct {
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      string          `json:"id"`
		JSONRPC string          `json:"jsonrpc"`
	}

	type JSONRPCResponse struct {
		ID      string          `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
		JSONRPC string          `json:"jsonrpc"`
	}

	tunnel := make(chan JSONRPCRequest)
	pending := sync.Map{}

	result.Mux.HandleFunc("/rpc/request", func(w http.ResponseWriter, r *http.Request) {
		log.Info("rpc request received")
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			w.WriteHeader(http.StatusOK)
			return
		}
		response := make(chan JSONRPCResponse, 0)
		pending.Store(req.ID, response)
		log.Info("rpc tunnel sending", "id", req.ID)
		tunnel <- req
		log.Info("rpc tunnel waiting", "id", req.ID)
		resp := <-response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	result.Mux.HandleFunc("/rpc/response", func(w http.ResponseWriter, r *http.Request) {
		log.Info("rpc response received")
		var resp JSONRPCResponse
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if ch, ok := pending.LoadAndDelete(resp.ID); ok {
			ch.(chan JSONRPCResponse) <- resp
			close(ch.(chan JSONRPCResponse))
		}
		w.WriteHeader(http.StatusOK)
		return
	})

	result.Mux.HandleFunc("/rpc/tunnel", func(w http.ResponseWriter, r *http.Request) {
		log.Info("rpc tunnel connected")
		defer log.Info("rpc tunnel disconnected")
		flusher := w.(http.Flusher)
		ctx := r.Context()
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		for {
			select {
			case <-ctx.Done():
				return
			case req, ok := <-tunnel:
				if !ok {
					return
				}
				err := json.NewEncoder(w).Encode(req)
				if err != nil {
					log.Error("failed to encode rpc request", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				flusher.Flush()
			}
		}
	})

	return result, nil
}

func (s *Server) Start(ctx context.Context, p *project.Project) error {
	log := slog.Default().With("service", "server")
	log.Info("starting")
	defer log.Info("server done")

	resource.Register(ctx, p, s.Rpc)
	aws.Register(ctx, p, s.Rpc)
	scrap.Register(ctx, p, s.Rpc)
	runtime.Register(ctx, p, s.Rpc)

	server := &http.Server{
		Handler: s.Mux,
	}
	server.Addr = fmt.Sprintf("0.0.0.0:%d", s.Port)
	log.Info("server", "addr", server.Addr)
	serverPath := resolveServerFile(p.PathConfig(), p.App().Stage)
	u, _ := url.Parse("http://" + server.Addr)
	os.WriteFile(serverPath, []byte(u.String()), 0644)
	defer os.Remove(serverPath)
	go server.ListenAndServe()

	keyPath := filepath.Join(global.CertPath(), "key.pem")
	certPath := filepath.Join(global.CertPath(), "cert.pem")
	if _, err := os.Stat(keyPath); err == nil {
		log.Info("https enabled")
		proxy := httputil.NewSingleHostReverseProxy(u)
		go http.ListenAndServeTLS(
			fmt.Sprintf("0.0.0.0:%d", s.Port+1000),
			certPath,
			keyPath,
			proxy,
		)
		if err != nil {
			log.Error("failed to start https server", "err", err)
			return err
		}
	}

	<-ctx.Done()
	log.Info("shutting down server")
	go server.Shutdown(ctx)
	return nil
}

func port() (int, error) {
	port := 13557
	for {
		if port == 65535 {
			return 0, fmt.Errorf("no port available")
		}
		listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			port++
			continue
		}
		defer listener.Close()
		return port, nil
	}
}

type HttpConn struct {
	io.Reader
	io.Writer
}

func (c *HttpConn) Close() error { return nil }

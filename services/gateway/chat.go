package main

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	zistauth "github.com/saidmashhud/zist/internal/auth"
)

// mountWSProxy mounts a transparent WebSocket proxy at /api/chat/* → chatURL.
// The gateway injects X-User-* headers (already done by propagateAuth middleware)
// before forwarding. HookLine uses these to authenticate the connection.
func mountWSProxy(r chi.Router, chatURL string) {
	upstream, err := url.Parse(chatURL)
	if err != nil {
		slog.Error("invalid CHAT_URL", "err", err)
		return
	}

	proxy := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Require auth — unauthenticated WebSocket connections are rejected.
		p := zistauth.FromContext(req.Context())
		if p == nil || p.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Determine if this is a WebSocket upgrade request.
		if !isWebSocketUpgrade(req) {
			http.Error(w, "WebSocket upgrade required", http.StatusUpgradeRequired)
			return
		}

		// Dial upstream (HookLine) TCP directly and relay.
		target := upstream.Host
		if !strings.Contains(target, ":") {
			target += ":80"
		}

		conn, err := net.Dial("tcp", target)
		if err != nil {
			slog.Warn("chat: upstream dial failed", "err", err)
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		defer conn.Close()

		// Hijack the client connection for raw TCP relay.
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijacking not supported", http.StatusInternalServerError)
			return
		}
		clientConn, buf, err := hj.Hijack()
		if err != nil {
			slog.Warn("chat: hijack failed", "err", err)
			return
		}
		defer clientConn.Close()

		// Forward the original request to upstream, with injected user headers.
		req2, _ := http.NewRequest(req.Method, req.URL.String(), buf)
		req2.Header = req.Header.Clone()
		req2.Header.Set("X-User-ID", p.UserID)
		req2.Header.Set("X-Tenant-ID", p.TenantID)
		req2.URL.Host = upstream.Host
		req2.URL.Scheme = upstream.Scheme
		if req2.URL.Scheme == "" {
			req2.URL.Scheme = "http"
		}

		if err := req2.Write(conn); err != nil {
			slog.Warn("chat: forward request failed", "err", err)
			return
		}

		// Bidirectional relay.
		done := make(chan struct{}, 2)
		relay := func(dst io.Writer, src io.Reader) {
			io.Copy(dst, src) //nolint:errcheck
			done <- struct{}{}
		}
		go relay(conn, clientConn)
		go relay(clientConn, conn)
		<-done
	})

	// zistauth.Middleware reads X-User-* headers (set by propagateAuth) into context,
	// then RequireAuth checks that context — both are needed here since the gateway
	// never runs zistauth.Middleware globally (that's a service-layer concern).
	chatMW := chi.Chain(zistauth.Middleware, zistauth.RequireAuth)
	r.With(chatMW...).Handle("/api/chat", proxy)
	r.With(chatMW...).Handle("/api/chat/*", proxy)

	slog.Info("chat WebSocket proxy registered", "upstream", chatURL)
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

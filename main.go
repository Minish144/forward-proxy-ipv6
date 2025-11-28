package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/elazarl/goproxy"
)

const listenAddr = "0.0.0.0:3128"

var (
	proxyUser        = ""
	proxyPass        = ""
	proxyUpstreamUrl = ""
)

func isValidAuth(authHeader string) bool {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return false
	}
	payload, err := base64.StdEncoding.DecodeString(authHeader[6:])
	if err != nil {
		return false
	}
	pair := strings.SplitN(string(payload), ":", 2)
	return len(pair) == 2 && pair[0] == proxyUser && pair[1] == proxyPass
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isValidAuth(r.Header.Get("Proxy-Authorization")) {
			w.Header().Set("Proxy-Authenticate", "Basic realm=\"Proxy Required\"")
			http.Error(w, "Proxy authentication required", http.StatusProxyAuthRequired)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func init() {
	proxyUser = os.Getenv("PROXY_USER")
	proxyPass = os.Getenv("PROXY_PASS")
	proxyUpstreamUrl = os.Getenv("PROXY_UPSTREAM_URL")
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	upstreamURL, err := url.Parse(proxyUpstreamUrl)
	if err != nil {
		log.Fatal("Invalid upstream proxy URL:", err)
	}

	upstreamURL.User = url.UserPassword(proxyUser, proxyPass)
	proxy.Tr.Proxy = http.ProxyURL(upstreamURL)

	handler := authMiddleware(proxy)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Forward proxy starting on %s (auth: %s:******)\n", listenAddr, proxyUser)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

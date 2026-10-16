package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// whoamiRouter mirrors the one decision that matters in main.go: which peers
// are allowed to set X-Forwarded-For.
func whoamiRouter(t *testing.T, trusted []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(trusted); err != nil {
		t.Fatalf("SetTrustedProxies(%v): %v", trusted, err)
	}
	router.GET("/whoami", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	return router
}

func clientIPFor(t *testing.T, router *gin.Engine, remoteAddr, forwardedFor string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Body.String()
}

// The whole point of the trusted-proxy list: a header from a peer that is not a
// proxy is a claim, not a fact. Before this was configured, gin believed the
// header from anyone, so allow_ips, approval_trusted_ips, the per-IP rate
// limits and the access log all recorded whatever the client typed.
func TestClientIPIgnoresForwardedForFromUntrustedPeer(t *testing.T) {
	router := whoamiRouter(t, []string{"127.0.0.1/32", "::1/128"})

	if got := clientIPFor(t, router, "203.0.113.9:41234", "198.51.100.77"); got != "203.0.113.9" {
		t.Fatalf("client ip = %q, want the socket address 203.0.113.9", got)
	}

	// A real proxy on this host is believed, which is the deployment the
	// default list exists for.
	if got := clientIPFor(t, router, "127.0.0.1:41234", "198.51.100.77"); got != "198.51.100.77" {
		t.Fatalf("client ip = %q, want the forwarded address from a trusted peer", got)
	}

	// `none` means nobody is a proxy - not even loopback.
	strict := whoamiRouter(t, nil)
	if got := clientIPFor(t, strict, "127.0.0.1:41234", "198.51.100.77"); got != "127.0.0.1" {
		t.Fatalf("client ip = %q, want the socket address when no proxy is trusted", got)
	}
}

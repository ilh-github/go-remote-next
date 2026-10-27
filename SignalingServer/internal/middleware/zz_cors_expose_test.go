package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// The export endpoints put the filename in Content-Disposition, and a
// cross-origin fetch cannot read that header unless the server exposes it. The
// download still works without it, but the client silently falls back to a name
// of its own and loses the server's timestamp -- which is the only thing that
// tells two exports taken minutes apart apart.
//
// This is not a hypothetical: the owner-side audit export was built and worked
// in a same-origin check while producing a default filename under the CORS
// configuration the server actually ships.
type staticOriginProvider struct{ origins []string }

func (p staticOriginProvider) GetAllowedOrigins() []string { return p.origins }
func (p staticOriginProvider) GetAPIKey() string           { return "" }

func TestCORSExposesContentDisposition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(staticOriginProvider{origins: []string{"https://console.example.com"}}))
	router.GET("/v1/me/audit-logs:export", func(c *gin.Context) {
		c.Header("Content-Disposition", `attachment; filename="my-audit-logs.csv"`)
		c.String(http.StatusOK, "id,action\n")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export", nil)
	req.Header.Set("Origin", "https://console.example.com")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	exposed := rec.Header().Get("Access-Control-Expose-Headers")
	if exposed == "" {
		t.Fatal("Access-Control-Expose-Headers is empty; a cross-origin client cannot read Content-Disposition")
	}
	for _, want := range []string{"Content-Disposition", "Content-Type"} {
		if !containsHeader(exposed, want) {
			t.Fatalf("exposed headers %q do not include %q", exposed, want)
		}
	}
}

// A request with no Origin is not a cross-origin request, so the middleware must
// not start emitting CORS headers it was not asked for.
func TestCORSWithoutAnOriginSendsNoExposeHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(staticOriginProvider{origins: []string{"https://console.example.com"}}))
	router.GET("/v1/me/audit-logs:export", func(c *gin.Context) {
		c.String(http.StatusOK, "id,action\n")
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me/audit-logs:export", nil))

	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want none for a same-origin request", got)
	}
}

func containsHeader(list, want string) bool {
	for _, part := range splitAndTrim(list, ",") {
		if part == want {
			return true
		}
	}
	return false
}

func splitAndTrim(list, sep string) []string {
	var out []string
	current := ""
	for _, r := range list {
		if string(r) == sep {
			if current != "" {
				out = append(out, current)
			}
			current = ""
			continue
		}
		if r != ' ' {
			current += string(r)
		}
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

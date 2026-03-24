package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestBuildWSRequestHeadersUsesAccessToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	headers := buildWSRequestHeaders(c, "svc.clawhost.svc.cluster.local:18789", "bot-secret")

	if got := headers.Get("Authorization"); got != "Bearer bot-secret" {
		t.Fatalf("expected proxy to inject bot access token, got %q", got)
	}
}

func TestBuildBackendQueryAddsAccessToken(t *testing.T) {
	got := buildBackendQuery("foo=bar", "bot-secret")
	if got != "foo=bar&token=bot-secret" {
		t.Fatalf("expected access token in backend query, got %q", got)
	}
}

func TestBuildBackendQueryPreservesExistingToken(t *testing.T) {
	got := buildBackendQuery("token=existing&foo=bar", "bot-secret")
	values, err := url.ParseQuery(got)
	if err != nil {
		t.Fatalf("expected valid query string, got error %v", err)
	}
	if values.Get("token") != "existing" {
		t.Fatalf("expected existing token to be preserved, got %q", values.Get("token"))
	}
	if values.Get("foo") != "bar" {
		t.Fatalf("expected other query params to be preserved, got %q", values.Get("foo"))
	}
}

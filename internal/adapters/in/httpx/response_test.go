package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponseHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("json error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		JSONError(c, http.StatusBadRequest, "invalid", "bad request")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "{\"code\":\"invalid\",\"error\":\"bad request\"}" {
			t.Fatalf("unexpected body: %s", body)
		}
	})

	t.Run("json field error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		JSONFieldError(c, http.StatusBadRequest, "invalid_field", "bad field", "cpf")

		if body := rec.Body.String(); body != "{\"code\":\"invalid_field\",\"error\":\"bad field\",\"field\":\"cpf\"}" {
			t.Fatalf("unexpected body: %s", body)
		}
	})

	t.Run("abort error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		AbortError(c, http.StatusUnauthorized, "unauthorized", "nope")

		if !c.IsAborted() {
			t.Fatalf("expected context to be aborted")
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})
}

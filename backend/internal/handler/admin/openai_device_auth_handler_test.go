package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIDeviceHandlersRequireOwnerAndValidBody(t *testing.T) {
	h := &OpenAIOAuthHandler{}
	for _, handler := range []gin.HandlerFunc{h.StartDeviceAuth, h.PollDeviceAuth, h.CancelDeviceAuth} {
		for _, authenticated := range []bool{false, true} {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/device", strings.NewReader("invalid-json"))
			ctx.Request.Header.Set("Content-Type", "application/json")
			if authenticated {
				ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
			}
			handler(ctx)
			if authenticated {
				require.Equal(t, 400, recorder.Code)
				require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
			} else {
				require.Equal(t, 401, recorder.Code)
			}
		}
	}
}

package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

func TestOpenAIDeviceProtocol(t *testing.T) {
	for _, payload := range []string{`{"device_auth_id":"device","user_code":"ABCD","interval":"5"}`, `{"device_auth_id":"device","usercode":"ABCD","interval":5}`} {
		t.Run(payload, func(t *testing.T) {
			server := newLocalTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]string
				if json.NewDecoder(r.Body).Decode(&body) != nil || r.Method != "POST" || body["client_id"] != openai.ClientID {
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(payload))
			}))
			defer server.Close()
			client := &openaiOAuthService{deviceAuthURL: server.URL}
			code, err := client.StartDeviceAuth(context.Background(), "")
			require.NoError(t, err)
			require.Equal(t, "ABCD", code.UserCode)
			require.Equal(t, 5, code.Interval)
		})
	}
	for _, status := range []int{403, 404, 200, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newLocalTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]string
				if json.NewDecoder(r.Body).Decode(&body) != nil || r.URL.Path != "/token" || body["device_auth_id"] != "device" || body["user_code"] != "ABCD" {
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"authorization_code":"code","code_verifier":"verifier"}`))
			}))
			defer server.Close()
			client := &openaiOAuthService{deviceAuthURL: server.URL}
			grant, err := client.PollDeviceAuth(context.Background(), "device", "ABCD", "")
			if status == 500 {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if status == 200 {
				require.Equal(t, "verifier", grant.CodeVerifier)
			} else {
				require.Nil(t, grant)
			}
		})
	}
}

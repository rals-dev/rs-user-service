package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-service/config"
	"user-service/constants"

	"github.com/gin-gonic/gin"
)

// TestExtractBearerToken locks in strict RFC 6750 "Bearer " prefix parsing:
// previously the code only checked strings.Contains(token, "Bearer"), which
// accepted malformed headers like "XBearer abc" or "SomeBearerToken abc".
func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"valid bearer token", "Bearer abc.def.ghi", "abc.def.ghi", false},
		{"empty header", "", "", true},
		{"missing prefix entirely", "abc.def.ghi", "", true},
		{"lowercase bearer rejected", "bearer abc.def.ghi", "", true},
		{"bearer as substring, not prefix", "XBearer abc.def.ghi", "", true},
		{"prefix with no token", "Bearer ", "", true},
		{"prefix with only whitespace", "Bearer    ", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractBearerToken(tc.header)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got token %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func newAPIKeyTestContext(apiKey, serviceName, requestAt string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(constants.XApiKey, apiKey)
	req.Header.Set(constants.XServiceName, serviceName)
	req.Header.Set(constants.XRequestAt, requestAt)
	c.Request = req
	return c
}

// TestValidateAPIKey_ConstantTimeCompare verifies the functional correctness
// of the signature check after switching from a plain "!=" string compare to
// crypto/subtle.ConstantTimeCompare: a matching signature must still pass, a
// mismatched one must still fail, and a non-hex api key must fail cleanly
// (not panic).
func TestValidateAPIKey_ConstantTimeCompare(t *testing.T) {
	config.Config.SignatureKey = "unit-test-signature-key"
	serviceName := "svc"
	requestAt := "2024-01-01T00:00:00Z"

	validateKey := fmt.Sprintf("%s:%s:%s", serviceName, config.Config.SignatureKey, requestAt)
	sum := sha256.Sum256([]byte(validateKey))
	validHash := hex.EncodeToString(sum[:])

	if err := validateAPIKey(newAPIKeyTestContext(validHash, serviceName, requestAt)); err != nil {
		t.Fatalf("expected matching signature to pass, got %v", err)
	}

	if err := validateAPIKey(newAPIKeyTestContext(hex.EncodeToString([]byte("wrong-signature-bytes!!")), serviceName, requestAt)); err == nil {
		t.Fatal("expected mismatched signature to fail")
	}

	if err := validateAPIKey(newAPIKeyTestContext("not-hex!!", serviceName, requestAt)); err == nil {
		t.Fatal("expected non-hex api key to fail without panicking")
	}
}

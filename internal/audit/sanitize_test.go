package audit

import (
	"encoding/json"
	"testing"
)

func TestSanitizeParams(t *testing.T) {
	params := map[string]any{
		"channel_id": "C123",
		"token":      "secret-value",
		"message":    string(make([]byte, 600)),
	}
	sanitized := SanitizeParams(params)
	var out map[string]any
	if err := json.Unmarshal(sanitized, &out); err != nil {
		t.Fatal(err)
	}
	if out["token"] != "[REDACTED]" {
		t.Fatalf("expected redacted token, got %#v", out["token"])
	}
	msg, ok := out["message"].(string)
	if !ok || len(msg) <= 500 {
		t.Fatalf("expected truncated message, got len=%d", len(msg))
	}
}

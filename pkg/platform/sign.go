// Package platform provides shared utilities for Synctacles platform API communication.
package platform

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// SignRequest adds an X-Signature header with HMAC-SHA256 of the body.
// If secret is empty, no signature is added (backwards-compatible).
func SignRequest(req *http.Request, body []byte, secret string) {
	if secret == "" {
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	req.Header.Set("X-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
}

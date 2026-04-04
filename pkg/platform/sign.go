// Package platform provides shared utilities for Synctacles platform API communication.
package platform

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// platformSecret is a shared HMAC key for signing requests to Synctacles Workers.
// It is intentionally embedded in the binary (not a true secret — apps are open-source).
// Purpose: block bots/scanners, verify requests originate from Synctacles apps.
const platformSecret = "b9faaf5f15915737626eda88dea836ac47e9d3d83d370bbef4233c5637d2e6f9"

// SignRequest adds an X-Signature header with HMAC-SHA256 of the body.
func SignRequest(req *http.Request, body []byte) {
	mac := hmac.New(sha256.New, []byte(platformSecret))
	mac.Write(body)
	req.Header.Set("X-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
}

package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// postWebhook delivers payload as JSON to url with up to three attempts and
// returns the last HTTP status code (0 when the request never completed).
// When secret is set the body is signed with HMAC-SHA256 and the digest is
// sent as `X-Webhook-Signature: sha256=<hex>`, the same scheme the admin
// webhooks use — receivers can share one verification routine.
func postWebhook(url, secret string, payload interface{}) int {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0
	}

	var lastStatus int
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return 0
		}
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastStatus = 0
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}
		resp.Body.Close()
		lastStatus = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	return lastStatus
}

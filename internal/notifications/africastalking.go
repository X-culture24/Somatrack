package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	atLiveURL    = "https://api.africastalking.com/version1/messaging"
	atSandboxURL = "https://api.sandbox.africastalking.com/version1/messaging"
)

var atHTTPClient = &http.Client{Timeout: 20 * time.Second}

// atResponse is the subset of Africa's Talking's bulk-SMS reply we act on.
// Each recipient carries its own status — the HTTP call can succeed while the
// message itself is rejected (bad number, no credit, unregistered sender ID).
type atResponse struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			Number     string `json:"number"`
			Status     string `json:"status"`
			StatusCode int    `json:"statusCode"`
			Cost       string `json:"cost"`
			MessageID  string `json:"messageId"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

// sendAfricasTalking posts one message. Returning an error lets the asynq
// worker mark the log row failed and retry.
func (s *SMSSender) sendAfricasTalking(ctx context.Context, to, message string) error {
	if s.APIKey == "" || s.Username == "" {
		return fmt.Errorf("africastalking: SMS_API_KEY and AFRICAS_TALKING_USERNAME must be set")
	}
	phone, err := NormalizeKenyanPhone(to)
	if err != nil {
		return err
	}
	endpoint := s.APIURL
	if endpoint == "" {
		endpoint = atLiveURL
		if s.Username == "sandbox" {
			endpoint = atSandboxURL
		}
	}

	form := url.Values{}
	form.Set("username", s.Username)
	form.Set("to", phone)
	form.Set("message", message)
	// Only send a sender ID when one is configured: an unregistered ID makes
	// Africa's Talking reject every message, while omitting it falls back to
	// the shared short code.
	if s.APISender != "" {
		form.Set("from", s.APISender)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("apiKey", s.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := atHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("africastalking: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("africastalking: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out atResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("africastalking: unreadable response: %w", err)
	}
	if len(out.SMSMessageData.Recipients) == 0 {
		return fmt.Errorf("africastalking: no recipients accepted: %s", out.SMSMessageData.Message)
	}
	for _, rc := range out.SMSMessageData.Recipients {
		// 100 Processed, 101 Sent, 102 Queued — everything else is a rejection.
		if rc.StatusCode < 100 || rc.StatusCode > 102 {
			return fmt.Errorf("africastalking: %s rejected: %s (%d)", rc.Number, rc.Status, rc.StatusCode)
		}
	}
	return nil
}

// NormalizeKenyanPhone turns the formats parents actually give the school
// (0712345678, 712345678, 254712345678, +254 712 345 678) into E.164, which
// Africa's Talking requires.
func NormalizeKenyanPhone(raw string) (string, error) {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch {
	case len(d) == 12 && strings.HasPrefix(d, "254"):
	case len(d) == 10 && strings.HasPrefix(d, "0"):
		d = "254" + d[1:]
	case len(d) == 9 && (d[0] == '7' || d[0] == '1'):
		d = "254" + d
	default:
		return "", fmt.Errorf("invalid Kenyan phone number %q", raw)
	}
	if d[3] != '7' && d[3] != '1' {
		return "", fmt.Errorf("invalid Kenyan mobile number %q", raw)
	}
	return "+" + d, nil
}

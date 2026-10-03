package notifications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeKenyanPhone(t *testing.T) {
	ok := map[string]string{
		"0712345678":        "+254712345678",
		"712345678":         "+254712345678",
		"254712345678":      "+254712345678",
		"+254 712 345 678":  "+254712345678",
		"0110 123 456":      "+254110123456",
		"(+254) 110-123456": "+254110123456",
	}
	for in, want := range ok {
		got, err := NormalizeKenyanPhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizeKenyanPhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "12345", "0212345678", "+1 555 123 4567", "25471234567"} {
		if got, err := NormalizeKenyanPhone(bad); err == nil {
			t.Errorf("NormalizeKenyanPhone(%q) = %q; want error", bad, got)
		}
	}
}

func atServer(t *testing.T, reply string, check func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
}

func TestAfricasTalkingSend(t *testing.T) {
	srv := atServer(t, `{"SMSMessageData":{"Message":"Sent to 1/1","Recipients":[{"number":"+254712345678","status":"Success","statusCode":101}]}}`,
		func(r *http.Request) {
			if r.Header.Get("apiKey") != "k" {
				t.Errorf("apiKey header = %q", r.Header.Get("apiKey"))
			}
			_ = r.ParseForm()
			if r.Form.Get("to") != "+254712345678" || r.Form.Get("username") != "school" {
				t.Errorf("form = %v", r.Form)
			}
			if _, set := r.Form["from"]; set {
				t.Errorf("from must be omitted when no sender ID is configured")
			}
		})
	defer srv.Close()

	s := &SMSSender{Backend: "africastalking", APIKey: "k", Username: "school", APIURL: srv.URL}
	if err := s.Send(context.Background(), "0712345678", "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestAfricasTalkingRecipientRejected(t *testing.T) {
	srv := atServer(t, `{"SMSMessageData":{"Message":"Sent to 0/1","Recipients":[{"number":"+254712345678","status":"InsufficientBalance","statusCode":405}]}}`, nil)
	defer srv.Close()

	s := &SMSSender{Backend: "africastalking", APIKey: "k", Username: "school", APIURL: srv.URL}
	err := s.Send(context.Background(), "0712345678", "hello")
	if err == nil || !strings.Contains(err.Error(), "InsufficientBalance") {
		t.Fatalf("want InsufficientBalance error, got %v", err)
	}
}

func TestUnknownBackendFails(t *testing.T) {
	s := &SMSSender{Backend: "django.core.mail.backends.console.EmailBackend"}
	if err := s.Send(context.Background(), "0712345678", "hello"); err == nil {
		t.Fatal("unknown backend must return an error, not pretend to send")
	}
}

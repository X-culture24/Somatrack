// Package jenga talks to Equity Bank's Jenga API (finserve.africa) and
// parses the Instant Payment Notifications (IPNs) Jenga POSTs whenever money
// lands in the school's Equity account — including Paybill 247247 payments,
// whose Safaricom C2B confirmations go to Equity rather than to us.
package jenga

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	sandboxBaseURL = "https://uat.finserve.africa"
	liveBaseURL    = "https://api.finserve.africa"
)

// Config holds the Jenga HQ credentials. Everything comes from env so that
// secrets never live in the repo; see .env.example for the variable names.
type Config struct {
	Env            string // "sandbox" or "live"
	MerchantCode   string
	ConsumerSecret string
	APIKey         string
	PrivateKeyPath string
	CountryCode    string // e.g. "KE"
	AccountNumber  string // school's Equity account, used for balance checks
}

func ConfigFromEnv() Config {
	return Config{
		Env:            getEnv("JENGA_ENV", "sandbox"),
		MerchantCode:   os.Getenv("JENGA_MERCHANT_CODE"),
		ConsumerSecret: os.Getenv("JENGA_CONSUMER_SECRET"),
		APIKey:         os.Getenv("JENGA_API_KEY"),
		PrivateKeyPath: os.Getenv("JENGA_PRIVATE_KEY_PATH"),
		CountryCode:    getEnv("JENGA_COUNTRY_CODE", "KE"),
		AccountNumber:  getEnv("JENGA_ACCOUNT_NUMBER", os.Getenv("BANK_ACCOUNT_NO")),
	}
}

// Missing lists the credential env vars that are still empty, so callers can
// report exactly what's left to fill in instead of failing opaquely.
func (c Config) Missing() []string {
	var out []string
	if c.MerchantCode == "" {
		out = append(out, "JENGA_MERCHANT_CODE")
	}
	if c.ConsumerSecret == "" {
		out = append(out, "JENGA_CONSUMER_SECRET")
	}
	if c.APIKey == "" {
		out = append(out, "JENGA_API_KEY")
	}
	if c.PrivateKeyPath == "" {
		out = append(out, "JENGA_PRIVATE_KEY_PATH")
	}
	return out
}

type Client struct {
	cfg        Config
	baseURL    string
	httpClient *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
	privKey     *rsa.PrivateKey
}

func NewClient(cfg Config) *Client {
	base := sandboxBaseURL
	if strings.EqualFold(cfg.Env, "live") || strings.EqualFold(cfg.Env, "production") {
		base = liveBaseURL
	}
	return &Client{cfg: cfg, baseURL: base, httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) Configured() bool { return len(c.cfg.Missing()) == 0 }

func (c *Client) Env() string { return c.cfg.Env }

func (c *Client) AccountNumber() string { return c.cfg.AccountNumber }

// accessToken authenticates with the merchant credentials and caches the
// bearer token until shortly before it expires.
func (c *Client) accessToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}
	if missing := c.cfg.Missing(); len(missing) > 0 {
		return "", fmt.Errorf("jenga not configured, missing: %s", strings.Join(missing, ", "))
	}
	body, _ := json.Marshal(map[string]string{
		"merchantCode":   c.cfg.MerchantCode,
		"consumerSecret": c.cfg.ConsumerSecret,
	})
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/authentication/api/v3/authenticate/merchant", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("jenga auth: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("jenga auth: status %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   string `json:"expiresIn"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("jenga auth: unexpected response: %s", string(raw))
	}
	c.token = out.AccessToken
	// expiresIn is an absolute timestamp in some Jenga versions and absent in
	// others; a conservative fixed TTL avoids depending on either format.
	c.tokenExpiry = time.Now().Add(10 * time.Minute)
	return c.token, nil
}

func (c *Client) loadKey() (*rsa.PrivateKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.privKey != nil {
		return c.privKey, nil
	}
	b, err := os.ReadFile(c.cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read jenga private key: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("jenga private key: no PEM block found")
	}
	var key *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		key = k
	} else if k8, err8 := x509.ParsePKCS8PrivateKey(block.Bytes); err8 == nil {
		rk, ok := k8.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("jenga private key: not an RSA key")
		}
		key = rk
	} else {
		return nil, fmt.Errorf("jenga private key: %v", err8)
	}
	c.privKey = key
	return key, nil
}

// Sign produces Jenga's "signature" header: base64(RSA-SHA256(data)), where
// data is the concatenation of the fields each endpoint's docs specify.
func (c *Client) Sign(data string) (string, error) {
	key, err := c.loadKey()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(data))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (c *Client) get(path, signData string, out interface{}) error {
	token, err := c.accessToken()
	if err != nil {
		return err
	}
	sig, err := c.Sign(signData)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("signature", sig)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("jenga GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jenga GET %s: status %d: %s", path, resp.StatusCode, string(raw))
	}
	return json.Unmarshal(raw, out)
}

// AccountBalance is used as the end-to-end credentials check: it exercises
// the token, the API key and the RSA signature in one call.
func (c *Client) AccountBalance() (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("/v3-apis/account-api/v3.0/accounts/balances/%s/%s", c.cfg.CountryCode, c.cfg.AccountNumber)
	err := c.get(path, c.cfg.CountryCode+c.cfg.AccountNumber, &out)
	return out, err
}

// EFTResult is the Jenga Send Money API response.
type EFTResult struct {
	TransactionRef string
	Status         string
	Raw            map[string]interface{}
}

// SendMoney sends money via Equity EFT (salary/bank-to-bank transfer). Used
// for payroll disbursement to staff with Equity or any Kenyan bank account.
// Signature data: merchantCode + orderDate + sourceAccountNumber + amount.
func (c *Client) SendMoney(
	sourceAccount, destBankCode, destAccount, destName string,
	amount float64,
	reference, description, orderDate string,
) (*EFTResult, error) {
	amountStr := fmt.Sprintf("%.2f", amount)
	signData := c.cfg.MerchantCode + orderDate + sourceAccount + amountStr
	sig, err := c.Sign(signData)
	if err != nil {
		return nil, fmt.Errorf("jenga sendmoney sign: %w", err)
	}
	payload := map[string]interface{}{
		"source": map[string]interface{}{
			"countryCode":   c.cfg.CountryCode,
			"name":          "St Mary's School Kabete",
			"accountNumber": sourceAccount,
		},
		"destination": map[string]interface{}{
			"type":          "bank",
			"countryCode":   c.cfg.CountryCode,
			"name":          destName,
			"bankCode":      destBankCode,
			"accountNumber": destAccount,
		},
		"transfer": map[string]interface{}{
			"type":         "EFT",
			"amount":       amountStr,
			"currencyCode": "KES",
			"reference":    reference,
			"date":         orderDate,
			"description":  description,
		},
	}
	raw, err := c.post("/v3-apis/transaction-api/v3.0/remittance/sendmoney", sig, payload)
	if err != nil {
		return nil, err
	}
	res := &EFTResult{Raw: raw}
	if s, ok := raw["status"].(string); ok {
		res.Status = s
	}
	if ref, ok := raw["transactionReference"].(string); ok {
		res.TransactionRef = ref
	}
	return res, nil
}

// RTBRequest is the Jenga "Request to Pay" (EazzyPay C2B) request.
type RTBRequest struct {
	RequestRef  string
	Status      string
	Description string
	Raw         map[string]interface{}
}

// RequestToPay sends an EazzyPay collection request to the customer's mobile
// wallet. The customer gets a push notification and approves on their phone.
// Signature data: merchantCode + orderDate + mobileNumber + amount.
func (c *Client) RequestToPay(
	customerRef, partnerRef, mobileNumber, walletName string,
	amount float64,
	narrative, orderDate string,
) (*RTBRequest, error) {
	amountStr := fmt.Sprintf("%.2f", amount)
	signData := c.cfg.MerchantCode + orderDate + mobileNumber + amountStr
	sig, err := c.Sign(signData)
	if err != nil {
		return nil, fmt.Errorf("jenga requesttopay sign: %w", err)
	}
	payload := map[string]interface{}{
		"customerRef":  customerRef,
		"partnerRef":   partnerRef,
		"mobileNumber": mobileNumber,
		"walletName":   walletName,
		"amount":       amountStr,
		"currencyCode": "KES",
		"narrative":    narrative,
	}
	raw, err := c.post("/v3-apis/transaction-api/v3.0/merchant/mobile-money/requests/collection", sig, payload)
	if err != nil {
		return nil, err
	}
	res := &RTBRequest{Raw: raw}
	if s, ok := raw["status"].(string); ok {
		res.Status = s
	}
	if ref, ok := raw["referenceNumber"].(string); ok {
		res.RequestRef = ref
	}
	if d, ok := raw["description"].(string); ok {
		res.Description = d
	}
	return res, nil
}

func (c *Client) post(path, sig string, payload interface{}) (map[string]interface{}, error) {
	token, err := c.accessToken()
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("signature", sig)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jenga POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jenga POST %s: status %d: %s", path, resp.StatusCode, string(raw))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("jenga POST %s: decode: %w", path, err)
	}
	return out, nil
}

func getEnv(k, fb string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fb
}

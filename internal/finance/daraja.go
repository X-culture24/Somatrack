package finance

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// DarajaB2CClient is a Go port of Safaricom Daraja's B2C (business-to-customer)
// payout API — the same one used for salary/refund/promotion disbursements that
// the mpesa_daraja_b2c/ Django prototype in this repo exercised. It is kept here
// instead of running a separate Python service so the whole stack stays Go.
type DarajaB2CClient struct {
	BaseURL             string
	ConsumerKey         string
	ConsumerSecret      string
	InitiatorName       string
	InitiatorShortcode  string
	InitiatorPassword   string
	CertificatePath     string
	ResultURL           string
	TimeoutURL          string
	DefaultCommandID    string
	httpClient          *http.Client

	tokenMu     sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func NewDarajaB2CClient() *DarajaB2CClient {
	env := getEnv("DARAJA_ENV", "sandbox")
	base := "https://sandbox.safaricom.co.ke"
	if env == "production" {
		base = "https://api.safaricom.co.ke"
	}
	return &DarajaB2CClient{
		BaseURL:            getEnv("DARAJA_BASE_URL", base),
		ConsumerKey:        os.Getenv("DARAJA_CONSUMER_KEY"),
		ConsumerSecret:     os.Getenv("DARAJA_CONSUMER_SECRET"),
		InitiatorName:      os.Getenv("DARAJA_INITIATOR_NAME"),
		InitiatorShortcode: os.Getenv("DARAJA_INITIATOR_SHORTCODE"),
		InitiatorPassword:  os.Getenv("DARAJA_INITIATOR_PASSWORD"),
		CertificatePath:    getEnv("DARAJA_CERTIFICATE_PATH", "certs/SandboxCertificate.cer"),
		ResultURL:          os.Getenv("DARAJA_B2C_RESULT_URL"),
		TimeoutURL:         os.Getenv("DARAJA_B2C_TIMEOUT_URL"),
		DefaultCommandID:   getEnv("DARAJA_B2C_COMMAND_ID", "SalaryPayment"),
		httpClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

func getEnv(k, fallback string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fallback
}

type DarajaAPIError struct {
	Message string
}

func (e *DarajaAPIError) Error() string { return e.Message }

func (c *DarajaB2CClient) getAccessToken(forceRefresh bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if !forceRefresh && c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}
	if c.ConsumerKey == "" || c.ConsumerSecret == "" {
		return "", &DarajaAPIError{"DARAJA_CONSUMER_KEY / DARAJA_CONSUMER_SECRET are not configured"}
	}

	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.ConsumerKey, c.ConsumerSecret)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &DarajaAPIError{fmt.Sprintf("failed to reach Daraja OAuth endpoint: %v", err)}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", &DarajaAPIError{fmt.Sprintf("Daraja OAuth request failed (%d): %s", resp.StatusCode, string(body))}
	}
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &data); err != nil || data.AccessToken == "" {
		return "", &DarajaAPIError{fmt.Sprintf("Daraja OAuth response missing access_token: %s", string(body))}
	}
	expiresIn, _ := strconv.Atoi(data.ExpiresIn)
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	safetyMargin := 60
	if expiresIn-safetyMargin < 30 {
		safetyMargin = expiresIn - 30
	}
	c.cachedToken = data.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn-safetyMargin) * time.Second)
	return c.cachedToken, nil
}

// securityCredential RSA/PKCS1v15-encrypts the initiator password with Safaricom's
// published public certificate (PEM or DER), matching Daraja's SecurityCredential spec.
func (c *DarajaB2CClient) securityCredential() (string, error) {
	if c.InitiatorPassword == "" {
		return "", &DarajaAPIError{"DARAJA_INITIATOR_PASSWORD is not set — cannot derive SecurityCredential"}
	}
	raw, err := os.ReadFile(c.CertificatePath)
	if err != nil {
		return "", &DarajaAPIError{fmt.Sprintf("could not read Daraja public certificate at %q: %v", c.CertificatePath, err)}
	}

	var certDER []byte
	if block, _ := pem.Decode(raw); block != nil {
		certDER = block.Bytes
	} else {
		certDER = raw
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return "", &DarajaAPIError{fmt.Sprintf("could not parse Daraja certificate: %v", err)}
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return "", &DarajaAPIError{"Daraja certificate does not contain an RSA public key"}
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(c.InitiatorPassword))
	if err != nil {
		return "", &DarajaAPIError{fmt.Sprintf("failed to encrypt security credential: %v", err)}
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

var msisdnDigits = regexp.MustCompile(`\D`)

// NormalizeMSISDN converts common Kenyan phone formats to the 2547XXXXXXXX /
// 2541XXXXXXXX form Daraja expects.
func NormalizeMSISDN(phone string) (string, error) {
	digits := msisdnDigits.ReplaceAllString(phone, "")
	switch {
	case len(digits) == 12 && digits[:3] == "254":
		return digits, nil
	case len(digits) == 10 && digits[0] == '0':
		return "254" + digits[1:], nil
	case len(digits) == 9 && (digits[0] == '7' || digits[0] == '1'):
		return "254" + digits, nil
	}
	return "", &DarajaAPIError{fmt.Sprintf("%q is not a recognisable Safaricom MSISDN", phone)}
}

type B2CSendResult struct {
	OriginatorConversationID string
	ConversationID           string
	ResponseCode             string
	ResponseDescription      string
	Raw                      map[string]interface{}
}

// SendPayment triggers a B2C payout. Success here only means Safaricom accepted
// the request — the real outcome lands later on the ResultURL/QueueTimeOutURL
// webhooks (see disbursement.go).
func (c *DarajaB2CClient) SendPayment(phoneNumber string, amount float64, remarks, occasion, commandID, originatorConversationID string) (*B2CSendResult, error) {
	msisdn, err := NormalizeMSISDN(phoneNumber)
	if err != nil {
		return nil, err
	}
	if commandID == "" {
		commandID = c.DefaultCommandID
	}
	cred, err := c.securityCredential()
	if err != nil {
		return nil, err
	}
	if len(remarks) > 100 {
		remarks = remarks[:100]
	}
	if len(occasion) > 100 {
		occasion = occasion[:100]
	}
	payload := map[string]interface{}{
		"OriginatorConversationID": originatorConversationID,
		"InitiatorName":            c.InitiatorName,
		"SecurityCredential":       cred,
		"CommandID":                commandID,
		"Amount":                   strconv.Itoa(int(amount)),
		"PartyA":                   c.InitiatorShortcode,
		"PartyB":                   msisdn,
		"Remarks":                  remarks,
		"QueueTimeOutURL":          c.TimeoutURL,
		"ResultURL":                c.ResultURL,
		"Occasion":                 occasion,
	}
	raw, err := c.post("/mpesa/b2c/v1/paymentrequest", payload, true)
	if err != nil {
		return nil, err
	}
	res := &B2CSendResult{
		OriginatorConversationID: originatorConversationID,
		Raw:                      raw,
	}
	if v, ok := raw["ConversationID"].(string); ok {
		res.ConversationID = v
	}
	if v, ok := raw["ResponseCode"]; ok {
		res.ResponseCode = fmt.Sprintf("%v", v)
	}
	if v, ok := raw["ResponseDescription"].(string); ok {
		res.ResponseDescription = v
	}
	return res, nil
}

func (c *DarajaB2CClient) post(path string, payload map[string]interface{}, retryOnAuthFailure bool) (map[string]interface{}, error) {
	token, err := c.getAccessToken(false)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(payload)

	do := func(tok string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		return c.httpClient.Do(req)
	}

	resp, err := do(token)
	if err != nil {
		return nil, &DarajaAPIError{fmt.Sprintf("failed to reach Daraja at %s: %v", path, err)}
	}
	if resp.StatusCode == http.StatusUnauthorized && retryOnAuthFailure {
		resp.Body.Close()
		token, err = c.getAccessToken(true)
		if err != nil {
			return nil, err
		}
		resp, err = do(token)
		if err != nil {
			return nil, &DarajaAPIError{fmt.Sprintf("failed to reach Daraja at %s: %v", path, err)}
		}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		data = map[string]interface{}{"raw": string(raw)}
	}
	if resp.StatusCode >= 400 {
		return nil, &DarajaAPIError{fmt.Sprintf("Daraja request failed (%d): %v", resp.StatusCode, data)}
	}
	return data, nil
}

package igloohome

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"gopkg.in/square/go-jose.v2/json"
)

type Client struct {
	LockIDs                map[string]string
	ApprovalEmail          string
	ApprovalLink           string
	AdditionalInstructions string

	httpClient     *http.Client
	otpVariance    map[string]int
	hourlyVariance map[string]int
}

type OTPRequestBody struct {
	Variance   int    `json:"variance"`
	StartDate  string `json:"startDate"`
	AccessName string `json:"accessName"`
}

type HourlyRequestBody struct {
	Variance   int    `json:"variance"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	AccessName string `json:"accessName"`
}

type TokenResponse struct {
	PIN string `json:"pin"`
	ID  string `json:"pinId"`
}

const otpURLPattern = "https://api.igloodeveloper.co/igloohome/devices/%s/algopin/onetime"
const hourlyURLPattern = "https://api.igloodeveloper.co/igloohome/devices/%s/algopin/hourly"

const tokenURL = "https://auth.igloohome.co/oauth2/token"

const maxOTPVariances = 5
const maxHourlyVariances = 3

func NewClient(clientID, clientSecret string, locks map[string]string) *Client {
	cc := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}
	return &Client{
		httpClient:     cc.Client(context.Background()),
		LockIDs:        locks,
		hourlyVariance: make(map[string]int),
		otpVariance:    make(map[string]int),
	}
}

func (c *Client) GenerateOTP(lock, name string) (string, error) {
	lockID, ok := c.LockIDs[lock]
	if !ok {
		return "", fmt.Errorf("no lock ID configured for %s", lock)
	}
	url := fmt.Sprintf(otpURLPattern, lockID)

	// API requires minute and second to be truncated to 0
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	otpReq := OTPRequestBody{
		Variance:   c.getOTPVariance(lockID),
		StartDate:  startDate.Format(time.RFC3339),
		AccessName: name,
	}

	return c.getToken(url, otpReq)
}

func (c *Client) GenerateHourly(lock, name string, duration time.Duration) (string, time.Time, error) {
	// API requires minute and second to be truncated to 0
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	endDate := startDate.Add(duration)

	lockID, ok := c.LockIDs[lock]
	if !ok {
		return "", endDate, fmt.Errorf("no lock ID configured for %s", lock)
	}
	url := fmt.Sprintf(hourlyURLPattern, lockID)

	otpReq := HourlyRequestBody{
		Variance:   c.getHourlyVariance(lockID),
		StartDate:  startDate.Format(time.RFC3339),
		EndDate:    endDate.Format(time.RFC3339),
		AccessName: name,
	}

	tok, err := c.getToken(url, otpReq)
	return tok, endDate, err
}

func (c *Client) getToken(url string, body any) (string, error) {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to build request body: %w", err)
	}

	fmt.Printf("%s", reqBody)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got invalid status code: %d, %s", resp.StatusCode, respBody)
	}

	var token TokenResponse
	err = json.Unmarshal(respBody, &token)
	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %w", err)
	}

	return token.PIN, nil
}

func (c *Client) getOTPVariance(lock string) int {
	if _, ok := c.otpVariance[lock]; !ok {
		c.otpVariance[lock] = 0
	}
	c.otpVariance[lock]++
	if c.otpVariance[lock] > maxOTPVariances {
		c.otpVariance[lock] = 1
	}
	return c.otpVariance[lock]
}

func (c *Client) getHourlyVariance(lock string) int {
	if _, ok := c.hourlyVariance[lock]; !ok {
		c.hourlyVariance[lock] = 0
	}
	c.hourlyVariance[lock]++
	if c.hourlyVariance[lock] > maxHourlyVariances {
		c.hourlyVariance[lock] = 1
	}
	return c.hourlyVariance[lock]
}

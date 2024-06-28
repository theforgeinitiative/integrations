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
	HTTPClient    *http.Client
	StorageLockID string

	otpVariance    int
	hourlyVariance int
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

func NewClient(clientID, clientSecret, lockID string) *Client {
	cc := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}
	return &Client{
		HTTPClient:    cc.Client(context.Background()),
		StorageLockID: lockID,
	}
}

func (c *Client) GenerateOTP(name string) (string, error) {
	url := fmt.Sprintf(otpURLPattern, c.StorageLockID)

	// API requires minute and second to be truncated to 0
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	otpReq := OTPRequestBody{
		Variance:   c.getOTPVariance(),
		StartDate:  startDate.Format(time.RFC3339),
		AccessName: name,
	}

	return c.getToken(url, otpReq)
}

func (c *Client) GenerateHourly(name string, duration time.Duration) (string, time.Time, error) {
	url := fmt.Sprintf(hourlyURLPattern, c.StorageLockID)

	// API requires minute and second to be truncated to 0
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	endDate := startDate.Add(duration)

	otpReq := HourlyRequestBody{
		Variance:   c.getHourlyVariance(),
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

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
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

func (c *Client) getOTPVariance() int {
	c.otpVariance++
	if c.otpVariance > maxOTPVariances {
		c.otpVariance = 1
	}
	return c.otpVariance
}

func (c *Client) getHourlyVariance() int {
	c.hourlyVariance++
	if c.hourlyVariance > maxHourlyVariances {
		c.hourlyVariance = 1
	}
	return c.hourlyVariance
}

package vatusa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrMissingEnv    = errors.New("missing environment variables")
	ErrInvalidStatus = errors.New("invalid status code returned")
)

type Client struct {
	url    string
	apiKey string
	hc     *http.Client
}

func NewClient(url, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		url:    url,
		apiKey: apiKey,
		hc:     httpClient,
	}
}

func (c *Client) FetchRoster(ctx context.Context) ([]Controller, error) {
	if c.apiKey == "" || c.url == "" {
		return nil, ErrMissingEnv
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/facility/ZAU/roster/both?apikey=%s&t=%d", c.url, c.apiKey, time.Now().UnixMilli()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	var vatusaData Roster

	err = json.NewDecoder(resp.Body).Decode(&vatusaData)
	if err != nil {
		return nil, err
	}

	return vatusaData.Data, nil
}

func (c *Client) FetchACE(ctx context.Context) ([]int, error) {
	if c.apiKey == "" || c.url == "" {
		return nil, ErrMissingEnv
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/user/roles/ZHQ/ACE?apikey=%s&t=%d", c.url, c.apiKey, time.Now().UnixMilli()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	var aceData ACERoster

	err = json.NewDecoder(resp.Body).Decode(&aceData)
	if err != nil {
		return nil, err
	}

	cids := make([]int, 0, len(aceData.Data))
	for _, member := range aceData.Data {
		cids = append(cids, member.CID)
	}

	return cids, nil
}

func (ct *CertSyncDate) UnmarshalJSON(byteSlice []byte) error {
	// The JSON data byte array is a quoted string, e.g., "2025-12-08 03:23:09"
	// We unmarshal it into a regular string variable.
	if string(byteSlice) == "null" {
		return nil
	}

	stringTime := string(bytes.Trim(byteSlice, "\""))

	timeTime, err := time.Parse("2006-01-02 15:04:05", stringTime)
	if err != nil {
		return err
	}

	*ct = CertSyncDate(timeTime)

	return nil
}

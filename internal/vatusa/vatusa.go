package vatusa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	vatusaURL     = "https://api.vatusa.net/v2/facility/ZAU/roster/both"
	ErrMissingEnv = errors.New("missing environment variables")
)

func FetchData(ctx context.Context) ([]Controller, error) {
	apiKey := strings.TrimSpace(os.Getenv("VATUSA_API_KEY"))
	if apiKey == "" {
		return []Controller{}, ErrMissingEnv
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?apikey=%s&t=%d", vatusaURL, apiKey, time.Now().UnixMilli()), nil)
	if err != nil {
		log.Printf("Error creating request for VATUSA data: %v\n", err)
		return []Controller{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error fetching VATUSA data: %v\n", err)
		return []Controller{}, err
	}
	defer resp.Body.Close()

	var vatusaData Roster

	err = json.NewDecoder(resp.Body).Decode(&vatusaData)
	if err != nil {
		log.Printf("Error unmarshaling VATUSA data: %v\n", err)
		return []Controller{}, err
	}

	return vatusaData.Data, nil
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

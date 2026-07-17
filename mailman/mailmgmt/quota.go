package mailmgmt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// Dovecot request struct
type DoveadmRequest struct {
	Command    string            `json:"command"`
	Parameters map[string]string `json:"parameters"`
}

type QuotaPayload struct {
	QuotaName string `json:"quota_name"`
	Type      string `json:"type"`    // Will be "STORAGE"
	Value     string `json:"value"`   // Current used bytes
	Limit     string `json:"limit"`   // Maximum bytes
	Percent   string `json:"percent"` // Usage percentage
}

// Dovecot response struct
type DoveadmResponse struct {
	Payload []QuotaPayload `json:"payload"`
}

// GetUserQuota reaches out to the Mailserver container and asks for the current quota
func GetUserQuota(addr string, emails []string, apiPassword string) (map[string]QuotaPayload, error) {

	requests := make([]DoveadmRequest, 0, len(emails))
	for _, email := range emails {
		requests = append(requests, DoveadmRequest{
			Command: "quotaGet",
			Parameters: map[string]string{
				"user": email,
			},
		})
	}

	jsonData, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/doveadm/v1"), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, ErrAPI.WithCause(err)
	}

	// Dovecot requires the Authorization header to be "X-Dovecot-API <base64(password)>"
	authHeader := "X-Dovecot-API " + base64.StdEncoding.EncodeToString([]byte(apiPassword))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrAPI.WithCause(
			fmt.Errorf("failed to connect to doveadm api: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrAPI.WithCause(
			fmt.Errorf("doveadm API returned status: %d", resp.StatusCode))
	}

	var dovecotResp []DoveadmResponse
	if err := json.NewDecoder(resp.Body).Decode(&dovecotResp); err != nil {
		return nil, ErrQuota.WithCause(err)
	}

	fmt.Println(resp)

	if len(dovecotResp) == 0 || len(dovecotResp[0].Payload) == 0 {
		return nil, ErrQuotaNotExists.WithCause(
			fmt.Errorf("no quota data found for user %v", emails))
	}

	return nil, nil
}

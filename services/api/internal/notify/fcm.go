package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/msriv/voyagami/services/api/internal/model"
)

// FCM sends through Firebase Cloud Messaging (HTTP v1). iOS goes through FCM
// too, which hands it to APNs. It uses the service's own identity, so no key
// is stored; that identity needs roles/firebasecloudmessaging.admin.
type FCM struct {
	Project string
	// BaseURL and Tokens are for tests; the zero values use Google's.
	BaseURL string
	Tokens  oauth2.TokenSource
	HTTP    *http.Client
}

// Send delivers one message.
func (f *FCM) Send(ctx context.Context, d model.Device, m Message) error {
	ts := f.Tokens
	if ts == nil {
		var err error
		ts, err = google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/firebase.messaging")
		if err != nil {
			return fmt.Errorf("push credentials: %w", err)
		}
	}
	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("push token: %w", err)
	}
	base := f.BaseURL
	if base == "" {
		base = "https://fcm.googleapis.com"
	}
	body, _ := json.Marshal(map[string]any{"message": map[string]any{
		"token":        d.PushToken,
		"notification": map[string]string{"title": m.Title, "body": m.Body},
		"data":         m.Data,
	}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/projects/"+f.Project+"/messages:send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tok.SetAuthHeader(req)
	client := f.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("push request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 == 2 {
		return nil
	}
	var e struct {
		Error struct {
			Status  string `json:"status"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&e)
	// A dead token answers 404 with status NOT_FOUND or UNREGISTERED.
	if e.Error.Status == "NOT_FOUND" || strings.Contains(e.Error.Status, "UNREGISTERED") || resp.StatusCode == http.StatusNotFound {
		return ErrInvalidToken
	}
	for _, det := range e.Error.Details {
		if det.ErrorCode == "UNREGISTERED" {
			return ErrInvalidToken
		}
	}
	// The body is not echoed: it can repeat the token.
	return fmt.Errorf("push service answered %d", resp.StatusCode)
}

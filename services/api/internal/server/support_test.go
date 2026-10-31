package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeSupport struct{ got []model.SupportMessage }

func (f *fakeSupport) Add(_ context.Context, m model.SupportMessage) (string, error) {
	f.got = append(f.got, m)
	return "m1", nil
}

func TestSendSupportMessage(t *testing.T) {
	f := &fakeSupport{}
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithSupport(f))
	body := `{"category":"bug","text":"  The map is blank  ","diagnostics":{"appVersion":"1.2.0","os":"ios","requestId":"r-1"}}`
	rec := call(h, "POST", "/v1/support/messages", "owner", body)
	if rec.Code != 202 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var r api.SupportReceipt
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Id != "m1" || len(f.got) != 1 || f.got[0].UserID != "u-owner" || f.got[0].Text != "The map is blank" || f.got[0].Diagnostics.OS != "ios" || f.got[0].Diagnostics.RequestID != "r-1" {
		t.Fatalf("stored %+v, receipt %+v", f.got, r)
	}

	for name, b := range map[string]string{
		"empty text":   `{"category":"bug","text":"   "}`,
		"too long":     `{"category":"bug","text":"` + strings.Repeat("a", model.MaxSupportText+1) + `"}`,
		"bad category": `{"category":"gossip","text":"hi"}`,
		"bad os":       `{"category":"bug","text":"hi","diagnostics":{"os":"windows"}}`,
	} {
		if c := call(h, "POST", "/v1/support/messages", "owner", b).Code; c != 400 {
			t.Errorf("%s: %d, want 400", name, c)
		}
	}
	if call(h, "POST", "/v1/support/messages", "", body).Code != 401 {
		t.Error("needs sign-in")
	}
	if len(f.got) != 1 {
		t.Errorf("bad messages were stored: %d", len(f.got))
	}
}

func TestSupportMessagesAreRateLimited(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"a": "u-a", "b": "u-b"}, WithSupport(&fakeSupport{}))
	body := `{"category":"other","text":"hi"}`
	for i := 0; i < 5; i++ {
		if c := call(h, "POST", "/v1/support/messages", "a", body).Code; c != 202 {
			t.Fatalf("message %d: %d", i+1, c)
		}
	}
	if c := call(h, "POST", "/v1/support/messages", "a", body).Code; c != 429 {
		t.Errorf("sixth message: %d, want 429", c)
	}
	if c := call(h, "POST", "/v1/support/messages", "b", body).Code; c != 202 {
		t.Errorf("another person is not limited: %d", c)
	}
}

func TestSupportOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "POST", "/v1/support/messages", "owner", `{"category":"bug","text":"x"}`).Code; c != 503 {
		t.Errorf("%d", c)
	}
}

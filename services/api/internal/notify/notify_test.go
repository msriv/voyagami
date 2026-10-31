package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/msriv/voyagami/services/api/internal/model"
)

type memDevices struct{ list []model.Device }

func (m *memDevices) Put(context.Context, string, model.Device) error { return nil }
func (m *memDevices) List(context.Context, string) ([]model.Device, error) {
	return append([]model.Device(nil), m.list...), nil
}
func (m *memDevices) Delete(_ context.Context, _, id string) error {
	for i, d := range m.list {
		if d.ID == id {
			m.list = append(m.list[:i], m.list[i+1:]...)
			break
		}
	}
	return nil
}

type fakeSender map[string]error

func (f fakeSender) Send(_ context.Context, d model.Device, _ Message) error { return f[d.ID] }

func TestSendToUserDropsDeadTokensAndKeepsGoing(t *testing.T) {
	devs := &memDevices{list: []model.Device{{ID: "ok"}, {ID: "dead"}, {ID: "down"}, {ID: "ok2"}}}
	boom := errors.New("boom")
	s := Service{Devices: devs, Sender: fakeSender{"dead": ErrInvalidToken, "down": boom}}
	sent, err := s.SendToUser(context.Background(), "u", Message{})
	if sent != 2 || !errors.Is(err, boom) {
		t.Fatalf("sent %d err %v", sent, err)
	}
	for _, d := range devs.list {
		if d.ID == "dead" {
			t.Fatal("dead token kept")
		}
	}
	if len(devs.list) != 3 {
		t.Fatalf("devices left: %v", devs.list)
	}
}

func TestLogRedacts(t *testing.T) {
	var buf bytes.Buffer
	l := Log{Logger: slog.New(slog.NewTextHandler(&buf, nil))}
	_ = l.Send(context.Background(), model.Device{Platform: "ios", PushToken: "secret-token-1234"},
		Message{Title: "Asha added a stop", Body: "Cafe Giang", Data: map[string]string{"kind": "stop_added", "place": "Cafe Giang"}})
	out := buf.String()
	for _, bad := range []string{"secret-token", "Asha", "Cafe Giang"} {
		if strings.Contains(out, bad) {
			t.Errorf("log leaks %q: %s", bad, out)
		}
	}
	if !strings.Contains(out, "1234") || !strings.Contains(out, "stop_added") {
		t.Errorf("log missing detail: %s", out)
	}
}

type staticToken struct{}

func (staticToken) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "t", TokenType: "Bearer"}, nil
}

func TestFCM(t *testing.T) {
	var got map[string]any
	status, answer := 200, `{}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		if r.URL.Path != "/v1/projects/proj/messages:send" || r.Header.Get("Authorization") != "Bearer t" {
			t.Errorf("bad request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	defer srv.Close()
	f := &FCM{Project: "proj", BaseURL: srv.URL, Tokens: staticToken{}}
	d := model.Device{PushToken: "dev-token"}
	m := Message{Title: "T", Body: "B", Data: map[string]string{"kind": "k"}}

	if err := f.Send(context.Background(), d, m); err != nil {
		t.Fatal(err)
	}
	msg := got["message"].(map[string]any)
	if msg["token"] != "dev-token" || msg["data"].(map[string]any)["kind"] != "k" {
		t.Fatalf("payload: %v", got)
	}

	status, answer = 404, `{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`
	if err := f.Send(context.Background(), d, m); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("dead token: %v", err)
	}

	status, answer = 500, `{"error":{"message":"dev-token exploded"}}`
	err := f.Send(context.Background(), d, m)
	if err == nil || errors.Is(err, ErrInvalidToken) || strings.Contains(err.Error(), "dev-token") {
		t.Fatalf("server error: %v", err)
	}
}

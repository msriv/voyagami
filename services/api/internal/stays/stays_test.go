package stays

import (
	"net/url"
	"strings"
	"testing"
)

const sample = `[{"id":"alpha","name":"Alpha Stays","baseUrl":"https://alpha.example/search",
 "params":{"destination":"q","checkIn":"in","checkOut":"out","guests":"adults"},
 "fixed":{"aid":"PLACEHOLDER"}}]`

func TestBuild(t *testing.T) {
	ps, err := Load(sample)
	if err != nil || len(ps) != 1 {
		t.Fatalf("load: %v %v", ps, err)
	}
	got, err := Build(ps[0], Search{Destination: "Hà Nội", CheckIn: "2026-12-01", CheckOut: "2026-12-04", Guests: 2})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	want := map[string]string{"q": "Hà Nội", "in": "2026-12-01", "out": "2026-12-04", "adults": "2", "aid": "PLACEHOLDER"}
	for k, v := range want {
		if u.Query().Get(k) != v {
			t.Errorf("%s = %q, want %q (url %s)", k, u.Query().Get(k), v, got)
		}
	}
	if u.Host != "alpha.example" || u.Path != "/search" {
		t.Errorf("address changed: %s", got)
	}
}

func TestBuildLeavesOutWhatIsMissing(t *testing.T) {
	ps, _ := Load(sample)
	got, _ := Build(ps[0], Search{Destination: "Hanoi"})
	if strings.Contains(got, "in=") || strings.Contains(got, "adults=") {
		t.Errorf("empty pieces must be left out: %s", got)
	}
}

func TestLoadRejectsBadConfig(t *testing.T) {
	for name, raw := range map[string]string{
		"not json":     `{`,
		"no id":        `[{"name":"A","baseUrl":"https://a.example"}]`,
		"duplicate id": `[{"id":"a","name":"A","baseUrl":"https://a.example"},{"id":"a","name":"B","baseUrl":"https://b.example"}]`,
		"plain http":   `[{"id":"a","name":"A","baseUrl":"http://a.example"}]`,
	} {
		if _, err := Load(raw); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if ps, err := Load(""); err != nil || len(ps) != 0 {
		t.Errorf("empty config: %v %v", ps, err)
	}
}

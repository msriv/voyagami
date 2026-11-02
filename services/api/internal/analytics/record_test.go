package analytics

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestRecords(t *testing.T) {
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	at := now.Add(-time.Minute)
	ev := []Event{{Name: "trip_created", At: at, Props: map[string]any{"solo": true, "destinations": 2.0}}}
	c := Context{SessionID: "s1", Platform: "ios", AppVersion: "1.2.0+45", AppLanguage: "en", OSVersion: "has spaces", Plan: "free", HomeRegion: "IN"}

	r := Records(ev, "u1", c, SourceApp, now)[0]
	if r.UID != "u1" || r.Platform != "ios" || r.AppVersion != "1.2.0+45" || r.HomeRegion != "IN" || r.Source != "app" {
		t.Errorf("context kept: %+v", r)
	}
	if r.OSVersion != "" {
		t.Errorf("free text in context is dropped: %q", r.OSVersion)
	}
	if r.EventTime != at.UnixMicro() || r.ReceivedTime != now.UnixMicro() || len(r.EventID) != 32 {
		t.Errorf("times and id: %+v", r)
	}
	var props map[string]any
	if err := json.Unmarshal([]byte(r.Props), &props); err != nil || props["solo"] != true || props["destinations"] != 2.0 {
		t.Errorf("props as JSON text: %q", r.Props)
	}

	anon := Records(ev, "", c, SourceServer, now)[0]
	if anon.UID != "" || anon.Props != "" || anon.Platform != "" || anon.Plan != "" {
		t.Errorf("without a uid nothing ties the event to a person: %+v", anon)
	}
}

// The JSON keys must be the BigQuery column names (infra/modules/analytics).
func TestMessagesUseColumnNames(t *testing.T) {
	msgs, err := Messages(Records([]Event{{Name: "app_open", At: time.Unix(1, 0)}}, "u1", Context{}, SourceApp, time.Unix(2, 0)))
	if err != nil || len(msgs) != 1 {
		t.Fatal(err)
	}
	b, _ := base64.StdEncoding.DecodeString(msgs[0].Data)
	var row map[string]any
	_ = json.Unmarshal(b, &row)
	for _, k := range []string{"eventId", "name", "eventTime", "receivedTime", "uid", "source"} {
		if _, ok := row[k]; !ok {
			t.Errorf("missing column %s in %s", k, b)
		}
	}
	if row["eventTime"] != 1e6 {
		t.Errorf("eventTime in microseconds: %v", row["eventTime"])
	}
	if _, ok := row["props"]; ok {
		t.Errorf("empty props are left out: %s", b)
	}
}

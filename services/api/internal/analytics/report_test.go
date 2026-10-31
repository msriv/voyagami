package analytics

import "testing"

func TestSummarize(t *testing.T) {
	rows := []Row{
		{UID: "a", Name: "app_open"}, {UID: "a", Name: "app_open"}, {UID: "b", Name: "app_installed"},
		{UID: "a", Name: "sign_in"}, {UID: "a", Name: "trip_created"},
		{UID: "a", Name: "paywall_shown", Moment: "bestOrder"}, {UID: "b", Name: "paywall_shown", Moment: "bestOrder"},
		{UID: "a", Name: "paywall_dismissed", Moment: "bestOrder"}, {UID: "b", Name: "paywall_converted", Moment: "bestOrder"},
		{UID: "b", Name: "paid"},
	}
	r := Summarize("2027-01-01", rows)
	byName := map[string]Count{}
	for _, c := range r.Events {
		byName[c.Name] = c
	}
	if c := byName["app_open"]; c.Events != 2 || c.People != 1 {
		t.Errorf("app_open %+v", c)
	}
	want := map[string]int{"install": 2, "sign_in": 1, "first_trip": 1, "invite_sent": 0, "paid": 1}
	for _, s := range r.Funnel {
		if w, ok := want[s.Step]; ok && s.People != w {
			t.Errorf("%s: %d people, want %d", s.Step, s.People, w)
		}
	}
	if len(r.Paywall) != 1 || r.Paywall[0] != (Moment{Moment: "bestOrder", Shown: 2, Dismissed: 1, Converted: 1}) {
		t.Errorf("paywall %+v", r.Paywall)
	}
}

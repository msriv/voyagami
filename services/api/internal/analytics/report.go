package analytics

import "sort"

// Row is one stored event, cut down to what the weekly review counts.
type Row struct {
	UID    string
	Name   string
	Day    string // YYYY-MM-DD
	Moment string // the paywall moment, when the event has one
}

// Count is how often an event happened and how many different people did it.
type Count struct {
	Name   string `json:"name"`
	Events int    `json:"events"`
	People int    `json:"people"`
}

// Step is one stage of the activation funnel.
type Step struct {
	Step   string `json:"step"`
	People int    `json:"people"`
}

// Moment is the paywall outcome for one moment.
type Moment struct {
	Moment    string `json:"moment"`
	Shown     int    `json:"shown"`
	Dismissed int    `json:"dismissed"`
	Converted int    `json:"converted"`
}

// Report is the weekly review.
type Report struct {
	Since   string   `json:"since"`
	Events  []Count  `json:"events"`
	Funnel  []Step   `json:"funnel"`
	Paywall []Moment `json:"paywall"`
}

// funnel lists the steps in order. A person counts for a step if they did any
// of its events in the period; the steps are not forced into order.
var funnel = []struct {
	step   string
	events []string
}{
	{"install", []string{"app_installed", "app_open"}},
	{"sign_in", []string{"sign_in"}},
	{"first_trip", []string{"trip_created"}},
	{"invite_sent", []string{"invite_sent"}},
	{"crew_joined", []string{"crew_joined"}},
	{"first_expense", []string{"expense_added"}},
	{"trial_start", []string{"trial_started"}},
	{"paid", []string{"paid", "subscription_started", "trip_pass_bought"}},
}

// Summarize counts rows into the report.
func Summarize(since string, rows []Row) Report {
	events := map[string]int{}
	people := map[string]map[string]bool{}
	moments := map[string]*Moment{}
	for _, r := range rows {
		events[r.Name]++
		if people[r.Name] == nil {
			people[r.Name] = map[string]bool{}
		}
		people[r.Name][r.UID] = true
		if r.Moment == "" {
			continue
		}
		m := moments[r.Moment]
		if m == nil {
			m = &Moment{Moment: r.Moment}
			moments[r.Moment] = m
		}
		switch r.Name {
		case "paywall_shown":
			m.Shown++
		case "paywall_dismissed":
			m.Dismissed++
		case "paywall_converted":
			m.Converted++
		}
	}
	rep := Report{Since: since, Events: []Count{}, Funnel: []Step{}, Paywall: []Moment{}}
	for name, n := range events {
		rep.Events = append(rep.Events, Count{Name: name, Events: n, People: len(people[name])})
	}
	sort.Slice(rep.Events, func(i, j int) bool { return rep.Events[i].Name < rep.Events[j].Name })
	for _, f := range funnel {
		who := map[string]bool{}
		for _, e := range f.events {
			for uid := range people[e] {
				who[uid] = true
			}
		}
		rep.Funnel = append(rep.Funnel, Step{Step: f.step, People: len(who)})
	}
	for _, m := range moments {
		rep.Paywall = append(rep.Paywall, *m)
	}
	sort.Slice(rep.Paywall, func(i, j int) bool { return rep.Paywall[i].Moment < rep.Paywall[j].Moment })
	return rep
}

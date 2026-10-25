package entitlements

import "testing"

func counts(f Feature, n int) map[Feature]int { return map[Feature]int{f: n} }

func TestCheck(t *testing.T) {
	free := New(Free())
	tests := []struct {
		name       string
		svc        *Service
		user       User
		trip       *Trip
		feature    Feature
		wantOK     bool
		wantLimit  int
		wantNeeds  bool
		wantUsedAt int
	}{
		{"free: 1 of 2 upcoming trips", free, User{Used: counts(UpcomingTrips, 1)}, nil, UpcomingTrips, true, 2, false, 1},
		{"free: 2 of 2 upcoming trips stops the third", free, User{Used: counts(UpcomingTrips, 2)}, nil, UpcomingTrips, false, 2, true, 2},
		{"plus: any number of upcoming trips", free, User{Plus: true, Used: counts(UpcomingTrips, 40)}, nil, UpcomingTrips, true, Unlimited, false, 40},
		{"a trip pass does not lift the person's trip count", free, User{Used: counts(UpcomingTrips, 2)}, &Trip{Pass: true}, UpcomingTrips, false, 2, true, 2},
		{"free: first best order is allowed", free, User{}, &Trip{Used: counts(BestOrder, 0)}, BestOrder, true, 1, false, 0},
		{"free: second best order on the trip is not", free, User{}, &Trip{Used: counts(BestOrder, 1)}, BestOrder, false, 1, true, 1},
		{"the free best order is per trip", free, User{}, &Trip{Used: counts(BestOrder, 0)}, BestOrder, true, 1, false, 0},
		{"best order unlocks for the crew when a member has plus", free, User{}, &Trip{MemberPlus: true, Used: counts(BestOrder, 3)}, BestOrder, true, Unlimited, false, 3},
		{"best order unlocks for the crew with a trip pass", free, User{}, &Trip{Pass: true, Used: counts(BestOrder, 3)}, BestOrder, true, Unlimited, false, 3},
		{"free: 5th forwarded booking is fine, 6th is not", free, User{}, &Trip{Used: counts(ForwardedBookings, 4)}, ForwardedBookings, true, 5, false, 4},
		{"free: 6th forwarded booking", free, User{}, &Trip{Used: counts(ForwardedBookings, 5)}, ForwardedBookings, false, 5, true, 5},
		{"free: 4th file", free, User{}, &Trip{Used: counts(Files, 3)}, Files, false, 3, true, 3},
		{"free: 31st photo", free, User{}, &Trip{Used: counts(JournalPhotos, 30)}, JournalPhotos, false, 30, true, 30},
		{"photos: another member's plus does not unlock them", free, User{}, &Trip{MemberPlus: true, Used: counts(JournalPhotos, 30)}, JournalPhotos, false, 30, true, 30},
		{"photos: a trip pass does", free, User{}, &Trip{Pass: true, Used: counts(JournalPhotos, 30)}, JournalPhotos, true, Unlimited, false, 30},
		{"free: 4th ask draft this month", free, User{Used: counts(AskDrafts, 3)}, nil, AskDrafts, false, 3, true, 3},
		{"free: second currency is closed", free, User{}, &Trip{}, MultiCurrency, false, 0, true, 0},
		{"second currency opens for the crew with one plus member", free, User{}, &Trip{MemberPlus: true}, MultiCurrency, true, Unlimited, false, 0},
		{"free: offline maps are closed", free, User{}, nil, OfflineMaps, false, 0, true, 0},
		{"offline maps are for the person with plus, not the crew", free, User{}, &Trip{MemberPlus: true}, OfflineMaps, false, 0, true, 0},
		{"plus: offline maps", free, User{Plus: true}, nil, OfflineMaps, true, Unlimited, false, 0},
		{"unknown feature is refused", free, User{Plus: true}, nil, Feature("teleport"), false, 0, false, 0},
		{"dev: everything is open", New(NoLimits()), User{Used: counts(UpcomingTrips, 99)}, nil, UpcomingTrips, true, Unlimited, false, 99},
		{"dev: closed flags are open too", New(NoLimits()), User{}, &Trip{}, MultiCurrency, true, Unlimited, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.svc.Check(tt.user, tt.trip, tt.feature)
			want := Decision{Allowed: tt.wantOK, Limit: tt.wantLimit, Used: tt.wantUsedAt, NeedsPlus: tt.wantNeeds}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestAlwaysFreeCannotBeClosed(t *testing.T) {
	closed := Limits{}
	for f := range scopes {
		closed[f] = 0 // the harshest table possible
	}
	svc := New(closed)
	for _, f := range []Feature{JoinTrip, SettleUp, MeetPoints, DataExport, OfflinePlan} {
		if d := svc.Check(User{}, &Trip{}, f); !d.Allowed || d.NeedsPlus {
			t.Errorf("%s was limited: %+v", f, d)
		}
	}
}

func TestPlansSpecValues(t *testing.T) {
	want := map[Feature]int{UpcomingTrips: 2, BestOrder: 1, ForwardedBookings: 5, Files: 3, JournalPhotos: 30, AskDrafts: 3}
	for f, n := range want {
		if Free()[f] != n {
			t.Errorf("Free %s = %d, Plans Spec says %d", f, Free()[f], n)
		}
	}
	for f, n := range Free() {
		if _, ok := scopes[f]; !ok {
			t.Errorf("%s has a limit but no scope", f)
		}
		_ = n
	}
}

func TestForEnv(t *testing.T) {
	if ForEnv("dev")[UpcomingTrips] != Unlimited {
		t.Error("dev should have no limits")
	}
	for _, env := range []string{"staging", "prod"} {
		if ForEnv(env)[UpcomingTrips] != 2 {
			t.Errorf("%s should use the Free values", env)
		}
	}
}

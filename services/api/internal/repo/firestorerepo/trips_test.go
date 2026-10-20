package firestorerepo

import (
	"errors"
	"sync"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func sampleTrip(start, end string) model.Trip {
	return model.Trip{
		ID:    model.NewID(),
		Title: "Vietnam",
		Destinations: []model.Destination{
			{Name: "Hanoi", CountryCode: "VN", TimeZone: "Asia/Ho_Chi_Minh", LocalCurrency: "VND"},
			{Name: "Ha Long", CountryCode: "VN", TimeZone: "Asia/Ho_Chi_Minh", LocalCurrency: "VND"},
			{Name: "Siem Reap", CountryCode: "KH", TimeZone: "Asia/Phnom_Penh", LocalCurrency: "KHR"},
		},
		Dates:              model.TripDates{Start: start, End: end},
		TimeZone:           "Asia/Ho_Chi_Minh",
		SettlementCurrency: "EUR",
		Status:             model.TripPlanned,
	}
}

func member(uid string, role model.Role) model.Member {
	return model.Member{UID: uid, Role: role, Status: model.MemberActive, DisplayName: "Person " + uid[:4]}
}

func newOwnedTrip(t *testing.T, s *Trips, start, end string) (model.Trip, string) {
	t.Helper()
	owner := model.NewID()
	tr, err := s.Create(testCtx(t), sampleTrip(start, end), member(owner, model.RoleOwner))
	if err != nil {
		t.Fatalf("Create trip: %v", err)
	}
	return tr, owner
}

func TestTripsCreateAndGet(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	in := sampleTrip("2026-11-21", "2026-11-29")
	in.Budget = &model.Budget{Scope: "group", Amount: model.Money{AmountMinor: 250000, Currency: "EUR"}}
	owner := model.NewID()
	// Callers cannot pick a different role for the creator.
	got, err := s.Create(ctx, in, member(owner, model.RoleViewer))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.OwnerUID != owner || len(got.MemberUIDs) != 1 || got.MemberUIDs[0] != owner {
		t.Errorf("owner mirror wrong: %+v", got)
	}
	if got.SettlementCurrency != "EUR" || got.TimeZone != "Asia/Ho_Chi_Minh" || got.Dates.Start != "2026-11-21" {
		t.Errorf("trip fields wrong: %+v", got)
	}
	if got.Budget == nil || got.Budget.Amount.AmountMinor != 250000 || got.Budget.Amount.Currency != "EUR" {
		t.Errorf("budget = %+v", got.Budget)
	}
	if len(got.Destinations) != 3 || got.Destinations[2].LocalCurrency != "KHR" {
		t.Errorf("destinations = %+v", got.Destinations)
	}
	if want := []string{"KH", "VN"}; len(got.DestinationCountries) != 2 || got.DestinationCountries[0] != want[0] || got.DestinationCountries[1] != want[1] {
		t.Errorf("destinationCountries = %v, want %v", got.DestinationCountries, want)
	}
	if got.Plan.Unlocked || got.Usage != (model.TripUsage{}) {
		t.Errorf("plan or usage not zero on a new trip: %+v %+v", got.Plan, got.Usage)
	}
	if got.CreatedAt.IsZero() {
		t.Error("createdAt not set")
	}

	m, err := s.Member(ctx, got.ID, owner)
	if err != nil {
		t.Fatalf("Member: %v", err)
	}
	if m.Role != model.RoleOwner || m.Status != model.MemberActive || m.JoinedAt.IsZero() {
		t.Errorf("owner member = %+v", m)
	}
}

func TestTripsCreateTwiceFailsAndLeavesNoOrphanMember(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	tr, owner := newOwnedTrip(t, s, "2026-11-21", "2026-11-29")
	other := model.NewID()
	_, err := s.Create(ctx, tr, member(other, model.RoleOwner))
	if !errors.Is(err, repo.ErrAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrAlreadyExists", err)
	}
	if _, err := s.Member(ctx, tr.ID, other); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("failed create left a member behind: %v", err)
	}
	if got, _ := s.Get(ctx, tr.ID); got.OwnerUID != owner {
		t.Errorf("owner changed to %q", got.OwnerUID)
	}
}

func TestTripsValidation(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	bad := sampleTrip("2026-11-29", "2026-11-21")
	bad.SettlementCurrency = "eur"
	bad.Destinations = nil
	bad.TimeZone = "Mars/Olympus"
	_, err := s.Create(ctx, bad, member(model.NewID(), model.RoleOwner))
	if !model.IsValidation(err) {
		t.Fatalf("error = %v, want validation error", err)
	}
	both := sampleTrip("2026-11-21", "2026-11-29")
	both.Dates.Flexible = &model.FlexibleDates{Days: 9}
	if _, err := s.Create(ctx, both, member(model.NewID(), model.RoleOwner)); !model.IsValidation(err) {
		t.Errorf("fixed+flexible dates error = %v, want validation error", err)
	}
	if _, err := s.Get(ctx, bad.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("invalid trip was stored: %v", err)
	}
}

func TestTripsFlexibleDatesRoundTrip(t *testing.T) {
	s := NewTrips(testClient(t))
	tr := sampleTrip("", "")
	tr.Status = model.TripDraft
	tr.Dates = model.TripDates{Flexible: &model.FlexibleDates{Days: 9, Month: "2026-11"}}
	got, err := s.Create(testCtx(t), tr, member(model.NewID(), model.RoleOwner))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Dates.Flexible == nil || got.Dates.Flexible.Days != 9 || got.Dates.Flexible.Month != "2026-11" || got.Dates.Start != "" {
		t.Errorf("dates = %+v", got.Dates)
	}
}

func TestTripsListForMember(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	me := model.NewID()
	create := func(tr model.Trip) model.Trip {
		got, err := s.Create(ctx, tr, member(me, model.RoleOwner))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	late := create(sampleTrip("2027-03-01", "2027-03-05"))
	early := create(sampleTrip("2026-11-21", "2026-11-29"))
	flexTrip := sampleTrip("", "")
	flexTrip.Dates = model.TripDates{Flexible: &model.FlexibleDates{Days: 4}}
	flexible := create(flexTrip)

	// A trip I am not on, and one I left.
	otherTrip, _ := newOwnedTrip(t, s, "2026-12-01", "2026-12-03")
	_ = otherTrip
	leftTrip, ownerOfLeft := newOwnedTrip(t, s, "2026-12-10", "2026-12-12")
	_ = ownerOfLeft
	if err := s.AddMember(ctx, leftTrip.ID, member(me, model.RoleEditor)); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMembership(ctx, leftTrip.ID, me, model.MemberLeft); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListForMember(ctx, me)
	if err != nil {
		t.Fatalf("ListForMember: %v", err)
	}
	var ids []string
	for _, tr := range got {
		ids = append(ids, tr.ID)
	}
	want := []string{early.ID, late.ID, flexible.ID}
	if len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Errorf("trips = %v, want %v (soonest first, flexible last, no left or foreign trips)", ids, want)
	}
	none, err := s.ListForMember(ctx, model.NewID())
	if err != nil || len(none) != 0 {
		t.Errorf("stranger sees trips: %v, %v", none, err)
	}
}

func TestTripsUpdateChangesOnlyPatchedFields(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	tr, _ := newOwnedTrip(t, s, "2026-11-21", "2026-11-29")

	title := "Vietnam and Cambodia"
	dests := []model.Destination{{Name: "Da Nang", CountryCode: "VN", TimeZone: "Asia/Ho_Chi_Minh"}}
	dates := model.TripDates{Start: "2026-11-22", End: "2026-11-30"}
	status := model.TripActive
	budget := model.Budget{Scope: "perPerson", Amount: model.Money{AmountMinor: 100000, Currency: "EUR"}}
	got, err := s.Update(ctx, tr.ID, model.TripPatch{Title: &title, Destinations: &dests, Dates: &dates, Status: &status, Budget: &budget})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Title != title || got.Status != model.TripActive || got.Dates.Start != "2026-11-22" || got.Budget == nil {
		t.Errorf("patched fields wrong: %+v", got)
	}
	if len(got.Destinations) != 1 || len(got.DestinationCountries) != 1 || got.DestinationCountries[0] != "VN" {
		t.Errorf("destinations or derived countries not updated: %+v %v", got.Destinations, got.DestinationCountries)
	}
	if got.SettlementCurrency != "EUR" || got.OwnerUID != tr.OwnerUID || len(got.MemberUIDs) != 1 {
		t.Errorf("untouched fields changed: %+v", got)
	}

	flex := model.TripDates{Flexible: &model.FlexibleDates{Days: 5}}
	got, err = s.Update(ctx, tr.ID, model.TripPatch{Dates: &flex})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dates.Start != "" || got.Dates.Flexible == nil {
		t.Errorf("switching to flexible dates kept the old fixed dates: %+v", got.Dates)
	}

	bad := ""
	if _, err := s.Update(ctx, tr.ID, model.TripPatch{Title: &bad}); !model.IsValidation(err) {
		t.Errorf("empty title error = %v, want validation error", err)
	}
	if _, err := s.Update(ctx, model.NewID(), model.TripPatch{Title: &title}); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing trip error = %v, want ErrNotFound", err)
	}
}

func TestTripsMembershipKeepsMirrorInStep(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	tr, owner := newOwnedTrip(t, s, "2026-11-21", "2026-11-29")
	friend := model.NewID()

	if err := s.AddMember(ctx, tr.ID, member(friend, model.RoleEditor)); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	got, _ := s.Get(ctx, tr.ID)
	if !contains(got.MemberUIDs, owner) || !contains(got.MemberUIDs, friend) {
		t.Errorf("memberUids = %v after add", got.MemberUIDs)
	}
	if err := s.AddMember(ctx, tr.ID, member(friend, model.RoleViewer)); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate add error = %v, want ErrAlreadyExists", err)
	}

	if err := s.SetMemberRole(ctx, tr.ID, friend, model.RoleViewer); err != nil {
		t.Fatalf("SetMemberRole: %v", err)
	}
	m, _ := s.Member(ctx, tr.ID, friend)
	if m.Role != model.RoleViewer {
		t.Errorf("role = %s, want viewer", m.Role)
	}

	if err := s.EndMembership(ctx, tr.ID, friend, model.MemberRemoved); err != nil {
		t.Fatalf("EndMembership: %v", err)
	}
	got, _ = s.Get(ctx, tr.ID)
	if contains(got.MemberUIDs, friend) {
		t.Errorf("memberUids still lists a removed member: %v", got.MemberUIDs)
	}
	m, _ = s.Member(ctx, tr.ID, friend)
	if m.Status != model.MemberRemoved || m.DisplayName == "" {
		t.Errorf("removed member should keep a document with status removed and a name: %+v", m)
	}
	if err := s.SetMemberRole(ctx, tr.ID, friend, model.RoleEditor); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("role change on a removed member error = %v, want ErrNotFound", err)
	}

	// Rejoining through a new invite reactivates the same document.
	if err := s.AddMember(ctx, tr.ID, member(friend, model.RoleViewer)); err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	got, _ = s.Get(ctx, tr.ID)
	if !contains(got.MemberUIDs, friend) {
		t.Errorf("memberUids = %v after rejoin", got.MemberUIDs)
	}

	all, err := s.Members(ctx, tr.ID)
	if err != nil || len(all) != 2 || all[0].UID != owner {
		t.Errorf("Members = %+v, %v (owner first, then friend)", all, err)
	}
}

func TestTripsOwnerRulesAndMissingTrip(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	tr, owner := newOwnedTrip(t, s, "2026-11-21", "2026-11-29")

	if err := s.AddMember(ctx, tr.ID, member(model.NewID(), model.RoleOwner)); !errors.Is(err, repo.ErrForbidden) {
		t.Errorf("adding a second owner error = %v, want ErrForbidden", err)
	}
	if err := s.SetMemberRole(ctx, tr.ID, owner, model.RoleViewer); !errors.Is(err, repo.ErrForbidden) {
		t.Errorf("demoting the owner error = %v, want ErrForbidden", err)
	}
	friend := model.NewID()
	_ = s.AddMember(ctx, tr.ID, member(friend, model.RoleEditor))
	if err := s.SetMemberRole(ctx, tr.ID, friend, model.RoleOwner); !errors.Is(err, repo.ErrForbidden) {
		t.Errorf("promoting to owner error = %v, want ErrForbidden", err)
	}
	if err := s.EndMembership(ctx, tr.ID, owner, model.MemberLeft); !errors.Is(err, repo.ErrForbidden) {
		t.Errorf("removing the owner error = %v, want ErrForbidden", err)
	}
	if err := s.EndMembership(ctx, tr.ID, friend, model.MemberActive); !model.IsValidation(err) {
		t.Errorf("ending with status active error = %v, want validation error", err)
	}
	if err := s.AddMember(ctx, model.NewID(), member(friend, model.RoleEditor)); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("joining a missing trip error = %v, want ErrNotFound", err)
	}
	if _, err := s.Member(ctx, tr.ID, model.NewID()); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("unknown member error = %v, want ErrNotFound", err)
	}
	if _, err := s.Member(ctx, tr.ID, ""); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("empty uid error = %v, want ErrNotFound", err)
	}
}

// Two devices join with the same invite at once: exactly one wins, and the
// mirror lists the person once.
func TestTripsConcurrentJoinAddsOnce(t *testing.T) {
	s := NewTrips(testClient(t))
	ctx := testCtx(t)
	tr, _ := newOwnedTrip(t, s, "2026-11-21", "2026-11-29")
	friend := model.NewID()

	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = s.AddMember(ctx, tr.ID, member(friend, model.RoleEditor))
		}()
	}
	wg.Wait()

	ok, dup := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, repo.ErrAlreadyExists):
			dup++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 || dup != 3 {
		t.Errorf("successes = %d, duplicates = %d; want 1 and 3", ok, dup)
	}
	got, _ := s.Get(ctx, tr.ID)
	count := 0
	for _, u := range got.MemberUIDs {
		if u == friend {
			count++
		}
	}
	if count != 1 {
		t.Errorf("memberUids lists the friend %d times: %v", count, got.MemberUIDs)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

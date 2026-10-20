package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func sampleUser() model.User {
	return model.User{
		ID:           model.NewID(),
		DisplayName:  "Asha",
		HomeRegion:   "DE",
		HomeCurrency: "EUR",
		Units:        model.Units{Distance: "km", Temperature: "c", Clock: "24h", FirstWeekday: 1},
		Language:     "de",
		FormatLocale: "de-DE",
		RegionSource: "device",
		TravelStyle:  model.TravelStyle{Interests: []string{"food", "hiking"}, Pace: "relaxed", Spend: "mid"},
	}
}

func TestUsersCreateGet(t *testing.T) {
	s := NewUsers(testClient(t))
	ctx := testCtx(t)
	want := sampleUser()
	want.HomeCity = &model.HomeCity{Name: "Berlin"}
	want.PaymentHandles = []model.PaymentHandle{{Type: "paypal", Value: "asha@example.com"}}

	got, err := s.Create(ctx, want)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != want.ID || got.HomeRegion != "DE" || got.HomeCurrency != "EUR" || got.Units != want.Units {
		t.Errorf("created user = %+v", got)
	}
	if got.HomeCity == nil || got.HomeCity.Name != "Berlin" {
		t.Errorf("homeCity = %+v", got.HomeCity)
	}
	if len(got.PaymentHandles) != 1 || got.PaymentHandles[0].Type != "paypal" {
		t.Errorf("paymentHandles = %+v", got.PaymentHandles)
	}
	if len(got.TravelStyle.Interests) != 2 {
		t.Errorf("travelStyle = %+v", got.TravelStyle)
	}
	if time.Since(got.CreatedAt) > time.Minute || got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("timestamps not set by the server: %v %v", got.CreatedAt, got.UpdatedAt)
	}

	again, err := s.Get(ctx, want.ID)
	if err != nil || again.DisplayName != "Asha" {
		t.Fatalf("Get = %+v, %v", again, err)
	}
}

func TestUsersCreateTwiceFails(t *testing.T) {
	s := NewUsers(testClient(t))
	ctx := testCtx(t)
	u := sampleUser()
	if _, err := s.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, u); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrAlreadyExists", err)
	}
}

func TestUsersGetMissing(t *testing.T) {
	s := NewUsers(testClient(t))
	if _, err := s.Get(testCtx(t), model.NewID()); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("Get error = %v, want ErrNotFound", err)
	}
	if _, err := s.Get(testCtx(t), ""); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("Get empty ID error = %v, want ErrNotFound", err)
	}
}

func TestUsersValidation(t *testing.T) {
	s := NewUsers(testClient(t))
	u := sampleUser()
	u.HomeCurrency = "euro"
	u.HomeRegion = "de"
	u.Units.Clock = "military"
	_, err := s.Create(testCtx(t), u)
	if !model.IsValidation(err) {
		t.Fatalf("Create error = %v, want a validation error", err)
	}
	var v *model.ValidationError
	if !errors.As(err, &v) || len(v.Problems) != 3 {
		t.Errorf("want all 3 problems reported together, got %v", err)
	}
	// Nothing was stored.
	if _, err := s.Get(testCtx(t), u.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("invalid user was stored: %v", err)
	}
}

func TestUsersUpdateChangesOnlyPatchedFields(t *testing.T) {
	s := NewUsers(testClient(t))
	ctx := testCtx(t)
	u := sampleUser()
	created, err := s.Create(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	region, currency := "IN", "INR"
	units := model.Units{Distance: "km", Temperature: "c", Clock: "12h", FirstWeekday: 0}
	source := "confirmed"
	now := time.Now().UTC().Truncate(time.Millisecond)
	got, err := s.Update(ctx, u.ID, model.UserPatch{
		HomeRegion: &region, HomeCurrency: &currency, Units: &units, RegionSource: &source, OnboardedAt: &now,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.HomeRegion != "IN" || got.HomeCurrency != "INR" || got.Units != units || got.RegionSource != "confirmed" {
		t.Errorf("patched fields wrong: %+v", got)
	}
	if got.OnboardedAt == nil || !got.OnboardedAt.Equal(now) {
		t.Errorf("onboardedAt = %v, want %v", got.OnboardedAt, now)
	}
	if got.DisplayName != "Asha" || got.Language != "de" || got.FormatLocale != "de-DE" {
		t.Errorf("untouched fields changed: %+v", got)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("createdAt changed: %v -> %v", created.CreatedAt, got.CreatedAt)
	}
	if !got.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updatedAt not advanced: %v -> %v", created.UpdatedAt, got.UpdatedAt)
	}
}

func TestUsersUpdateRejectsBadPatchAndMissingUser(t *testing.T) {
	s := NewUsers(testClient(t))
	ctx := testCtx(t)
	bad := "usd"
	if _, err := s.Update(ctx, model.NewID(), model.UserPatch{HomeCurrency: &bad}); !model.IsValidation(err) {
		t.Errorf("bad patch error = %v, want validation error", err)
	}
	name := "x"
	if _, err := s.Update(ctx, model.NewID(), model.UserPatch{DisplayName: &name}); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing user error = %v, want ErrNotFound", err)
	}
}

func TestUsersMoneyAndLocaleAreNotHardCoded(t *testing.T) {
	// A US-style user round-trips with no India defaults sneaking in.
	s := NewUsers(testClient(t))
	u := sampleUser()
	u.HomeRegion, u.HomeCurrency, u.Language, u.FormatLocale = "US", "USD", "en", "en-US"
	u.Units = model.Units{Distance: "mi", Temperature: "f", Clock: "12h", FirstWeekday: 0}
	got, err := s.Create(testCtx(t), u)
	if err != nil {
		t.Fatal(err)
	}
	if got.HomeRegion != "US" || got.HomeCurrency != "USD" || got.Units.Distance != "mi" || got.Units.Temperature != "f" {
		t.Errorf("user = %+v", got)
	}
}

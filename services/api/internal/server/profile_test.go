package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeProfile struct {
	gotID    string
	gotPatch model.UserPatch
	user     model.User
	getErr   error
}

func (f *fakeProfile) Get(context.Context, string) (model.User, error) { return f.user, f.getErr }

func (f *fakeProfile) Update(_ context.Context, id string, p model.UserPatch) (model.User, error) {
	f.gotID, f.gotPatch = id, p
	if err := p.Validate(); err != nil {
		return model.User{}, err
	}
	return model.User{
		ID: id, DisplayName: "Asha", HomeRegion: "VN", HomeCurrency: "VND", RegionSource: "confirmed",
		Units:       model.Units{Distance: "km", Temperature: "c", Clock: "24h", FirstWeekday: 1},
		TravelStyle: model.TravelStyle{Interests: []string{"beaches"}, Pace: "slow"},
	}, nil
}

func patchMe(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUpdateMe(t *testing.T) {
	verifier := fakeVerifier{token: "good", id: auth.Identity{UserID: "u_1"}}
	t.Run("saves travel style for the token's user", func(t *testing.T) {
		f := &fakeProfile{}
		rec := patchMe(NewHandler(quiet(), verifier, WithProfile(f)), `{"id":"someone-else","travelStyle":{"interests":["beaches"],"pace":"slow"}}`)
		if rec.Code != 200 {
			t.Fatalf("status %d: %s", rec.Code, rec.Body)
		}
		if f.gotID != "u_1" || f.gotPatch.TravelStyle == nil || f.gotPatch.TravelStyle.Pace != "slow" {
			t.Errorf("got id %q patch %+v", f.gotID, f.gotPatch)
		}
		if !strings.Contains(rec.Body.String(), `"beaches"`) {
			t.Errorf("body %s", rec.Body)
		}
	})
	t.Run("saves home region, currency and units", func(t *testing.T) {
		f := &fakeProfile{}
		rec := patchMe(NewHandler(quiet(), verifier, WithProfile(f)), `{"homeRegion":"VN","homeCurrency":"VND","regionSource":"confirmed","units":{"distance":"km","temperature":"c","clock":"24h","firstWeekday":1}}`)
		if rec.Code != 200 {
			t.Fatalf("status %d: %s", rec.Code, rec.Body)
		}
		p := f.gotPatch
		if p.HomeRegion == nil || *p.HomeRegion != "VN" || *p.HomeCurrency != "VND" || *p.RegionSource != "confirmed" || p.Units == nil || p.Units.Clock != "24h" {
			t.Errorf("patch %+v", p)
		}
		if !strings.Contains(rec.Body.String(), `"homeCurrency":"VND"`) || !strings.Contains(rec.Body.String(), `"firstWeekday":1`) {
			t.Errorf("body %s", rec.Body)
		}
	})
	t.Run("bad region is 400", func(t *testing.T) {
		rec := patchMe(NewHandler(quiet(), verifier, WithProfile(&fakeProfile{})), `{"homeRegion":"vn"}`)
		if rec.Code != 400 {
			t.Errorf("status %d: %s", rec.Code, rec.Body)
		}
	})
	t.Run("bad pace is 400", func(t *testing.T) {
		rec := patchMe(NewHandler(quiet(), verifier, WithProfile(&fakeProfile{})), `{"travelStyle":{"pace":"zoom"}}`)
		if rec.Code != 400 {
			t.Errorf("status %d: %s", rec.Code, rec.Body)
		}
	})
	t.Run("no profile storage is 503", func(t *testing.T) {
		rec := patchMe(NewHandler(quiet(), verifier), `{}`)
		if rec.Code != 503 {
			t.Errorf("status %d", rec.Code)
		}
	})
	t.Run("signed out is 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		NewHandler(quiet(), verifier, WithProfile(&fakeProfile{})).ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Errorf("status %d", rec.Code)
		}
	})
}

func TestGetMeWithProfile(t *testing.T) {
	verifier := fakeVerifier{token: "good", id: auth.Identity{UserID: "u_1", DisplayName: "Token Name"}}
	get := func(f *fakeProfile) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("Authorization", "Bearer good")
		rec := httptest.NewRecorder()
		NewHandler(quiet(), verifier, WithProfile(f)).ServeHTTP(rec, req)
		return rec
	}
	rec := get(&fakeProfile{user: model.User{ID: "u_1", DisplayName: "Asha", HomeRegion: "IN", HomeCurrency: "INR"}})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"homeRegion":"IN"`) {
		t.Errorf("stored profile: %d %s", rec.Code, rec.Body)
	}
	rec = get(&fakeProfile{getErr: repo.ErrNotFound})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Token Name") {
		t.Errorf("missing profile should fall back to the token: %d %s", rec.Code, rec.Body)
	}
	if rec = get(&fakeProfile{getErr: errors.New("boom")}); rec.Code != 500 {
		t.Errorf("storage failure: %d", rec.Code)
	}
}

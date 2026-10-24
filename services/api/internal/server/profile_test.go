package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeProfile struct {
	gotID    string
	gotPatch model.UserPatch
}

func (f *fakeProfile) Update(_ context.Context, id string, p model.UserPatch) (model.User, error) {
	f.gotID, f.gotPatch = id, p
	if err := p.Validate(); err != nil {
		return model.User{}, err
	}
	return model.User{ID: id, DisplayName: "Asha", TravelStyle: model.TravelStyle{Interests: []string{"beaches"}, Pace: "slow"}}, nil
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

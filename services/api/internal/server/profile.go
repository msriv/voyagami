package server

import (
	"context"
	"errors"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
)

// Profile is the profile storage the routes need. repo.Users implements it.
type Profile interface {
	Update(ctx context.Context, id string, patch model.UserPatch) (model.User, error)
}

// WithProfile turns on profile changes. Without it they answer 503.
func WithProfile(p Profile) Option { return func(h *handlers) { h.profile = p } }

// UpdateMe changes the signed-in person's own profile. The ID always comes from
// the token, never from the request.
func (h *handlers) UpdateMe(ctx context.Context, req api.UpdateMeRequestObject) (api.UpdateMeResponseObject, error) {
	id, ok := auth.FromContext(ctx)
	if !ok {
		return nil, errors.New("UpdateMe reached without an identity")
	}
	if h.profile == nil {
		return api.UpdateMe503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, "Profiles are not available right now."))}, nil
	}
	var patch model.UserPatch
	if req.Body != nil {
		patch.DisplayName = req.Body.DisplayName
		if s := req.Body.TravelStyle; s != nil {
			patch.TravelStyle = &model.TravelStyle{}
			for _, i := range deref(s.Interests) {
				patch.TravelStyle.Interests = append(patch.TravelStyle.Interests, string(i))
			}
			if s.Pace != nil {
				patch.TravelStyle.Pace = string(*s.Pace)
			}
			if s.Spend != nil {
				patch.TravelStyle.Spend = string(*s.Spend)
			}
		}
	}
	user, err := h.profile.Update(ctx, id.UserID, patch)
	if err != nil {
		var invalid *model.ValidationError
		if errors.As(err, &invalid) {
			return api.UpdateMe400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, strings.Join(invalid.Problems, ". ")+"."))}, nil
		}
		return nil, err
	}
	return api.UpdateMe200JSONResponse(meFrom(user)), nil
}

func deref[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}

func meFrom(u model.User) api.Me {
	me := api.Me{Id: u.ID}
	if u.DisplayName != "" {
		me.DisplayName = &u.DisplayName
	}
	s := u.TravelStyle
	if len(s.Interests) > 0 || s.Pace != "" || s.Spend != "" {
		style := api.TravelStyle{}
		if len(s.Interests) > 0 {
			interests := make([]api.TravelStyleInterests, len(s.Interests))
			for i, v := range s.Interests {
				interests[i] = api.TravelStyleInterests(v)
			}
			style.Interests = &interests
		}
		if s.Pace != "" {
			pace := api.TravelStylePace(s.Pace)
			style.Pace = &pace
		}
		if s.Spend != "" {
			spend := api.TravelStyleSpend(s.Spend)
			style.Spend = &spend
		}
		me.TravelStyle = &style
	}
	return me
}

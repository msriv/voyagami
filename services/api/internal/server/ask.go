package server

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/ask"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// AskCounter counts a person's Ask drafts for the month.
type AskCounter interface {
	AddAskDraft(ctx context.Context, uid string, now time.Time) (int, error)
}

// WithAsk turns on Ask drafts. It also needs the catalogue, stops and trips.
func WithAsk(d ask.Drafter, c AskCounter) Option {
	return func(h *handlers) { h.asker, h.askCount = d, c }
}

const (
	msgNoAsk       = "Ask is not available right now."
	msgAskErr      = "We couldn't make a draft right now. Try again in a moment."
	msgAskLimit    = "The Free plan includes three Ask drafts a month. Plus lifts that."
	askRadius      = 10000
	askCandidates  = 60
	askDefaultLang = "en"
)

// CreateAskDraft asks the model to arrange nearby catalogue places into days.
// Nothing is saved and nothing is added to the trip.
func (h *handlers) CreateAskDraft(ctx context.Context, req api.CreateAskDraftRequestObject) (api.CreateAskDraftResponseObject, error) {
	id, err := identity(ctx, "CreateAskDraft")
	if err != nil {
		return nil, err
	}
	if h.asker == nil || h.catalog == nil || h.stops == nil || h.trips == nil {
		return api.CreateAskDraft503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoAsk))}, nil
	}
	role, err := h.myMembership(ctx, req.TripId, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return api.CreateAskDraft404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	if !model.Can(role, model.ActionEdit) {
		return api.CreateAskDraft403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	if req.Body == nil {
		return api.CreateAskDraft400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Tell us what you want.")}, nil
	}
	prompt := strings.TrimSpace(req.Body.Prompt)
	if prompt == "" || utf8.RuneCountInString(prompt) > ask.MaxPrompt {
		return api.CreateAskDraft400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "The request must be 1 to 1000 characters.")}, nil
	}
	lang := trimmed(req.Body.Lang)
	if lang == "" {
		lang = askDefaultLang
	}

	if over, err := h.overLimit(ctx, id.UserID, "", entitlements.AskDrafts, msgAskLimit); err != nil {
		return nil, err
	} else if over != nil {
		return api.CreateAskDraft403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(*over)}, nil
	}

	lat, lng, ok, err := h.tripCentre(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return api.CreateAskDraft400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Add a place with a position to the trip first.")}, nil
	}
	found, _, err := h.catalog.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: askRadius, Limit: askCandidates})
	if err != nil {
		return nil, err
	}
	empty := api.CreateAskDraft200JSONResponse(api.AskDraft{Days: []api.AskDraftDay{}})
	if len(found) == 0 {
		return h.withRemaining(ctx, id.UserID, empty, 0), nil
	}
	offered := make([]ask.Place, len(found))
	names := make(map[string]string, len(found))
	for i, n := range found {
		name := n.DisplayName(lang)
		offered[i] = ask.Place{ID: n.ID, Name: name, Group: string(n.Group), Tags: n.Tags}
		names[n.ID] = name
	}

	start := time.Now()
	raw, usage, err := h.asker.Draft(ctx, ask.Input{Prompt: prompt, Lang: lang, Places: offered})
	// Cost and latency only. The prompt and the draft never go in a log.
	slog.Default().Info("ask draft", "requestId", requestIDFrom(ctx), "ms", time.Since(start).Milliseconds(),
		"inputTokens", usage.InputTokens, "outputTokens", usage.OutputTokens, "failed", err != nil)
	if err != nil {
		slog.Default().Error("ask draft failed", "requestId", requestIDFrom(ctx), "error", err)
		return api.CreateAskDraft503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgAskErr))}, nil
	}

	out := api.AskDraft{Days: []api.AskDraftDay{}}
	for _, d := range ask.Clean(offered, raw) {
		day := api.AskDraftDay{Day: d.Day, Items: []api.AskDraftItem{}}
		for _, it := range d.Items {
			item := api.AskDraftItem{PlaceId: it.PlaceID, Name: names[it.PlaceID]}
			if it.Note != "" {
				note := it.Note
				item.Note = &note
			}
			day.Items = append(day.Items, item)
		}
		out.Days = append(out.Days, day)
	}
	if len(out.Days) == 0 {
		return h.withRemaining(ctx, id.UserID, api.CreateAskDraft200JSONResponse(out), 0), nil
	}
	total, err := h.askCount.AddAskDraft(ctx, id.UserID, time.Now())
	if err != nil {
		return nil, err
	}
	return h.withRemaining(ctx, id.UserID, api.CreateAskDraft200JSONResponse(out), total), nil
}

// withRemaining fills in how many drafts are left this month. total is the
// month's count after this draft, or 0 when nothing was counted.
func (h *handlers) withRemaining(ctx context.Context, uid string, r api.CreateAskDraft200JSONResponse, total int) api.CreateAskDraftResponseObject {
	left := entitlements.Unlimited
	if h.plans != nil && h.planSource != nil {
		if user, err := h.planSource.User(ctx, uid); err == nil {
			if d := h.plans.Check(user, nil, entitlements.AskDrafts); d.Limit != entitlements.Unlimited {
				// The plan source read the count before this draft was counted.
				used := d.Used
				if total > 0 {
					used = total
				}
				left = max(d.Limit-used, 0)
			}
		}
	}
	r.Remaining = &left
	return r
}

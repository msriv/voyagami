package server

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithMeetPoints turns on meet points and location sharing. They also need WithTrips.
func WithMeetPoints(m repo.MeetPoints, p repo.Presence) Option {
	return func(h *handlers) { h.meetPoints, h.presence = m, p }
}

const msgNoMeet = "Meeting up is not available right now."

// freeShareMinutes is how long anyone can share their location. Longer needs
// the live crew map (Plus).
const freeShareMinutes = 60

func (h *handlers) meetOff() bool { return h.trips == nil || h.meetPoints == nil || h.presence == nil }

func unavailableMeet(ctx context.Context) api.UnavailableJSONResponse {
	return api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoMeet))
}

func noSuchMeet(ctx context.Context) api.NotFoundJSONResponse {
	return api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such meet point."))
}

func meetFrom(m model.MeetPoint, uid string, now time.Time) api.MeetPoint {
	out := api.MeetPoint{
		Id: m.ID, Name: m.Name, Place: placeTo(m.Place), CreatedBy: m.CreatedBy, EndsAt: m.EndsAt(),
		ConfirmedBy: m.ConfirmedBy, IAmComing: slices.Contains(m.ConfirmedBy, uid), Status: api.MeetPointStatusActive,
	}
	if out.ConfirmedBy == nil {
		out.ConfirmedBy = []string{}
	}
	if !m.At.IsZero() {
		t := m.At
		out.At = &t
	}
	if !now.Before(m.EndsAt()) {
		out.Status = api.MeetPointStatusEnded
	}
	return out
}

// ListMeetPoints returns the trip's meet points.
func (h *handlers) ListMeetPoints(ctx context.Context, req api.ListMeetPointsRequestObject) (api.ListMeetPointsResponseObject, error) {
	if h.meetOff() {
		return api.ListMeetPoints503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "ListMeetPoints", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListMeetPoints404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	all, err := h.meetPoints.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := api.MeetPointList{MeetPoints: make([]api.MeetPoint, len(all))}
	for i, m := range all {
		out.MeetPoints[i] = meetFrom(m, uid, now)
	}
	return api.ListMeetPoints200JSONResponse(out), nil
}

// CreateMeetPoint proposes a place and time. Owners and editors only.
func (h *handlers) CreateMeetPoint(ctx context.Context, req api.CreateMeetPointRequestObject) (api.CreateMeetPointResponseObject, error) {
	if h.meetOff() {
		return api.CreateMeetPoint503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "CreateMeetPoint", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.CreateMeetPoint404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionEdit) {
		return api.CreateMeetPoint403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	b := req.Body
	m := model.MeetPoint{ID: model.NewID(), Name: b.Name, Place: placeFrom(b.Place), CreatedBy: uid}
	if b.At != nil {
		m.At = *b.At
	}
	created, err := h.meetPoints.Create(ctx, req.TripId, m)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.CreateMeetPoint400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.CreateMeetPoint201JSONResponse(meetFrom(created, uid, time.Now())), nil
}

// UpdateMeetPoint changes a meet point. Owners and editors only.
func (h *handlers) UpdateMeetPoint(ctx context.Context, req api.UpdateMeetPointRequestObject) (api.UpdateMeetPointResponseObject, error) {
	if h.meetOff() {
		return api.UpdateMeetPoint503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "UpdateMeetPoint", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.UpdateMeetPoint404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionEdit) {
		return api.UpdateMeetPoint403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	if _, err := h.meetPoints.Get(ctx, req.TripId, req.MeetPointId); errors.Is(err, repo.ErrNotFound) {
		return api.UpdateMeetPoint404JSONResponse{NotFoundJSONResponse: noSuchMeet(ctx)}, nil
	} else if err != nil {
		return nil, err
	}
	b := req.Body
	patch := model.MeetPointPatch{Name: b.Name, At: b.At}
	if b.Place != nil {
		pl := placeFrom(*b.Place)
		patch.Place = &pl
	}
	updated, err := h.meetPoints.Update(ctx, req.TripId, req.MeetPointId, patch)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.UpdateMeetPoint400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.UpdateMeetPoint200JSONResponse(meetFrom(updated, uid, time.Now())), nil
}

// DeleteMeetPoint removes a meet point. Owners and editors only.
func (h *handlers) DeleteMeetPoint(ctx context.Context, req api.DeleteMeetPointRequestObject) (api.DeleteMeetPointResponseObject, error) {
	if h.meetOff() {
		return api.DeleteMeetPoint503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	_, role, deny, err := h.crewRole(ctx, "DeleteMeetPoint", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.DeleteMeetPoint404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionEdit) {
		return api.DeleteMeetPoint403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	if err := h.meetPoints.Delete(ctx, req.TripId, req.MeetPointId); err != nil {
		return nil, err
	}
	return api.DeleteMeetPoint204Response{}, nil
}

func (h *handlers) setComing(ctx context.Context, op, tripID, meetPointID string, coming bool) (api.MeetPoint, any, error) {
	uid, _, deny, err := h.crewRole(ctx, op, tripID)
	if err != nil || deny != nil {
		return api.MeetPoint{}, deny, err
	}
	m, err := h.meetPoints.SetConfirmed(ctx, tripID, meetPointID, uid, coming)
	if errors.Is(err, repo.ErrNotFound) {
		return api.MeetPoint{}, noSuchMeet(ctx), nil
	}
	if err != nil {
		return api.MeetPoint{}, nil, err
	}
	return meetFrom(m, uid, time.Now()), nil, nil
}

// ConfirmMeetPoint says the caller is coming.
func (h *handlers) ConfirmMeetPoint(ctx context.Context, req api.ConfirmMeetPointRequestObject) (api.ConfirmMeetPointResponseObject, error) {
	if h.meetOff() {
		return api.ConfirmMeetPoint503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	m, deny, err := h.setComing(ctx, "ConfirmMeetPoint", req.TripId, req.MeetPointId, true)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ConfirmMeetPoint404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	return api.ConfirmMeetPoint200JSONResponse(m), nil
}

// UnconfirmMeetPoint says the caller is not coming.
func (h *handlers) UnconfirmMeetPoint(ctx context.Context, req api.UnconfirmMeetPointRequestObject) (api.UnconfirmMeetPointResponseObject, error) {
	if h.meetOff() {
		return api.UnconfirmMeetPoint503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	m, deny, err := h.setComing(ctx, "UnconfirmMeetPoint", req.TripId, req.MeetPointId, false)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.UnconfirmMeetPoint404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	return api.UnconfirmMeetPoint200JSONResponse(m), nil
}

func sharedFrom(p model.Presence) api.SharedLocation {
	return api.SharedLocation{Uid: p.UID, Lat: p.Lat, Lng: p.Lng, AccuracyM: p.AccuracyM, MeetPointId: strPtr(p.MeetPointID), UpdatedAt: p.UpdatedAt, EndsAt: p.EndsAt}
}

// ListSharedLocations returns who is sharing a location right now.
func (h *handlers) ListSharedLocations(ctx context.Context, req api.ListSharedLocationsRequestObject) (api.ListSharedLocationsResponseObject, error) {
	if h.meetOff() {
		return api.ListSharedLocations503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	_, _, deny, err := h.crewRole(ctx, "ListSharedLocations", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListSharedLocations404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	all, err := h.presence.List(ctx, req.TripId, time.Now())
	if err != nil {
		return nil, err
	}
	out := api.SharedLocationList{Locations: make([]api.SharedLocation, len(all))}
	for i, p := range all {
		out.Locations[i] = sharedFrom(p)
	}
	return api.ListSharedLocations200JSONResponse(out), nil
}

// ShareMyLocation starts or updates the caller's own sharing. Sharing ends by
// itself: updates keep the end time unless the caller sends a new duration.
func (h *handlers) ShareMyLocation(ctx context.Context, req api.ShareMyLocationRequestObject) (api.ShareMyLocationResponseObject, error) {
	if h.meetOff() {
		return api.ShareMyLocation503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "ShareMyLocation", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ShareMyLocation404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	bad := func(msg string) (api.ShareMyLocationResponseObject, error) {
		return api.ShareMyLocation400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	b := req.Body
	now := time.Now().UTC().Truncate(time.Microsecond)
	cur, err := h.presence.Get(ctx, req.TripId, uid)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	sharing := err == nil && now.Before(cur.EndsAt)

	var endsAt time.Time
	switch {
	case b.DurationMinutes != nil:
		if *b.DurationMinutes < 1 || *b.DurationMinutes > model.MaxShareMinutes {
			return bad("durationMinutes must be between 1 and 1440.")
		}
		endsAt = now.Add(time.Duration(*b.DurationMinutes) * time.Minute)
	case sharing:
		endsAt = cur.EndsAt
	default:
		endsAt = now.Add(model.DefaultShareMinutes * time.Minute)
	}
	// Only a new duration can go past the free hour; keeping the end time was already checked when it was set.
	if b.DurationMinutes != nil && *b.DurationMinutes > freeShareMinutes {
		if ok, err := h.featureAllowed(ctx, uid, req.TripId, entitlements.LiveCrewMap); err != nil {
			return nil, err
		} else if !ok {
			return api.ShareMyLocation403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgNeedsPlus))}, nil
		}
	}
	var meetID string
	if b.MeetPointId != nil && *b.MeetPointId != "" {
		m, err := h.meetPoints.Get(ctx, req.TripId, *b.MeetPointId)
		if errors.Is(err, repo.ErrNotFound) {
			return bad("meetPointId is not a meet point on this trip.")
		}
		if err != nil {
			return nil, err
		}
		if !now.Before(m.EndsAt()) {
			return api.ShareMyLocation409JSONResponse{ConflictJSONResponse: api.ConflictJSONResponse(errBody(ctx, codeConflict, "That meet is over."))}, nil
		}
		meetID = m.ID
		if m.EndsAt().Before(endsAt) {
			endsAt = m.EndsAt()
		}
	}
	p := model.Presence{UID: uid, Lat: b.Lat, Lng: b.Lng, AccuracyM: intOr0(b.AccuracyM), MeetPointID: meetID, UpdatedAt: now, EndsAt: endsAt}
	if err := h.presence.Put(ctx, req.TripId, p); err != nil {
		if msg, ok := badRequest(err); ok {
			return bad(msg)
		}
		return nil, err
	}
	return api.ShareMyLocation200JSONResponse(sharedFrom(p)), nil
}

// StopSharingMyLocation removes the caller's shared location.
func (h *handlers) StopSharingMyLocation(ctx context.Context, req api.StopSharingMyLocationRequestObject) (api.StopSharingMyLocationResponseObject, error) {
	if h.meetOff() {
		return api.StopSharingMyLocation503JSONResponse{UnavailableJSONResponse: unavailableMeet(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "StopSharingMyLocation", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.StopSharingMyLocation404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if err := h.presence.Delete(ctx, req.TripId, uid); err != nil {
		return nil, err
	}
	return api.StopSharingMyLocation204Response{}, nil
}

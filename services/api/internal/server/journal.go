package server

import (
	"context"
	"errors"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// JournalStore is the journal storage the routes need. repo.Journal implements it.
type JournalStore = repo.Journal

// WithJournal turns on the journal routes. They also need WithTrips.
func WithJournal(s JournalStore) Option { return func(h *handlers) { h.journal = s } }

const msgNoJournal = "The journal is not available right now."

func (h *handlers) journalOff() bool { return h.trips == nil || h.journal == nil }

func journalFrom(e model.JournalEntry) api.JournalEntry {
	out := api.JournalEntry{
		Date: e.Date, AuthorUid: e.AuthorUID, Visibility: api.JournalEntryVisibility(e.Visibility), Text: e.Text,
		StopIds: e.StopIDs,
	}
	if out.StopIds == nil {
		out.StopIds = []string{}
	}
	out.Mood = strPtr(e.Mood)
	return out
}

// journalAccess checks the caller is on the trip and, for writes, not a viewer.
func (h *handlers) journalAccess(ctx context.Context, op, tripID string, write bool) (uid string, deny any, err error) {
	id, err := identity(ctx, op)
	if err != nil {
		return "", nil, err
	}
	role, err := h.myMembership(ctx, tripID, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return "", api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip.")), nil
	}
	if err != nil {
		return "", nil, err
	}
	if write && role == model.RoleViewer {
		return "", errForbidden(ctx), nil
	}
	return id.UserID, nil, nil
}

// ListJournal returns crew entries and the caller's own private ones.
func (h *handlers) ListJournal(ctx context.Context, req api.ListJournalRequestObject) (api.ListJournalResponseObject, error) {
	if h.journalOff() {
		return api.ListJournal503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoJournal))}, nil
	}
	uid, deny, err := h.journalAccess(ctx, "ListJournal", req.TripId, false)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListJournal404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	all, err := h.journal.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.JournalList{Entries: []api.JournalEntry{}}
	for _, e := range all {
		if e.Visibility == model.VisibilityPrivate && e.AuthorUID != uid {
			continue
		}
		out.Entries = append(out.Entries, journalFrom(e))
	}
	return api.ListJournal200JSONResponse(out), nil
}

// PutJournalEntry saves the caller's entry for a day.
func (h *handlers) PutJournalEntry(ctx context.Context, req api.PutJournalEntryRequestObject) (api.PutJournalEntryResponseObject, error) {
	if h.journalOff() {
		return api.PutJournalEntry503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoJournal))}, nil
	}
	uid, deny, err := h.journalAccess(ctx, "PutJournalEntry", req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.PutJournalEntry404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.PutJournalEntry403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	b := req.Body
	e := model.JournalEntry{Date: req.Date, AuthorUID: uid, Visibility: model.VisibilityCrew, Text: b.Text}
	if b.Visibility != nil {
		e.Visibility = string(*b.Visibility)
	}
	if b.Mood != nil {
		e.Mood = string(*b.Mood)
	}
	if b.StopIds != nil {
		e.StopIDs = *b.StopIds
	}
	saved, err := h.journal.Put(ctx, req.TripId, e)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.PutJournalEntry400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.PutJournalEntry200JSONResponse(journalFrom(saved)), nil
}

// DeleteJournalEntry removes the caller's own entry for a day.
func (h *handlers) DeleteJournalEntry(ctx context.Context, req api.DeleteJournalEntryRequestObject) (api.DeleteJournalEntryResponseObject, error) {
	if h.journalOff() {
		return api.DeleteJournalEntry503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoJournal))}, nil
	}
	uid, deny, err := h.journalAccess(ctx, "DeleteJournalEntry", req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.DeleteJournalEntry404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.DeleteJournalEntry403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	if !model.ValidDate(req.Date) {
		return api.DeleteJournalEntry404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such day."))}, nil
	}
	if err := h.journal.Delete(ctx, req.TripId, req.Date, uid); err != nil {
		return nil, err
	}
	return api.DeleteJournalEntry204Response{}, nil
}

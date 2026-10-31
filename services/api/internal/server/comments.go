package server

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithComments turns on comments. They also need WithTrips; comments on a
// stop also need WithStops.
func WithComments(s repo.Comments) Option { return func(h *handlers) { h.comments = s } }

const msgNoComments = "Comments are not available right now."

func (h *handlers) commentsOff() bool { return h.trips == nil || h.comments == nil }

func unavailableComments(ctx context.Context) api.UnavailableJSONResponse {
	return api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoComments))
}

func commentFrom(c model.Comment) api.Comment {
	out := api.Comment{
		Id: c.ID, TargetType: api.CommentTargetType(c.TargetType), TargetId: c.TargetID, AuthorUid: c.AuthorUID,
		Body: c.Body, Mentions: c.Mentions, CreatedAt: c.CreatedAt,
	}
	if out.Mentions == nil {
		out.Mentions = []string{}
	}
	if !c.EditedAt.IsZero() {
		t := c.EditedAt
		out.EditedAt = &t
	}
	return out
}

// checkMentions returns a message if a mention is not an active member of the trip.
func (h *handlers) checkMentions(ctx context.Context, tripID string, mentions []string) (string, error) {
	seen := map[string]bool{}
	for _, uid := range mentions {
		if seen[uid] {
			return "each person can be mentioned once.", nil
		}
		seen[uid] = true
		if _, err := h.myMembership(ctx, tripID, uid); errors.Is(err, repo.ErrNotFound) {
			return "mentions must be people on this trip.", nil
		} else if err != nil {
			return "", err
		}
	}
	return "", nil
}

// ListComments returns the trip's comments, or one thread's.
func (h *handlers) ListComments(ctx context.Context, req api.ListCommentsRequestObject) (api.ListCommentsResponseObject, error) {
	if h.commentsOff() {
		return api.ListComments503JSONResponse{UnavailableJSONResponse: unavailableComments(ctx)}, nil
	}
	_, _, deny, err := h.crewRole(ctx, "ListComments", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListComments404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	p := req.Params
	if (p.TargetType == nil) != (p.TargetId == nil) {
		return api.ListComments400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Give both targetType and targetId, or neither.")}, nil
	}
	all, err := h.comments.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.CommentList{Comments: []api.Comment{}}
	for _, c := range all {
		if p.TargetType != nil && (c.TargetType != string(*p.TargetType) || c.TargetID != *p.TargetId) {
			continue
		}
		out.Comments = append(out.Comments, commentFrom(c))
	}
	return api.ListComments200JSONResponse(out), nil
}

// CreateComment adds a comment to a stop or the trip. Any member can.
func (h *handlers) CreateComment(ctx context.Context, req api.CreateCommentRequestObject) (api.CreateCommentResponseObject, error) {
	if h.commentsOff() {
		return api.CreateComment503JSONResponse{UnavailableJSONResponse: unavailableComments(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "CreateComment", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.CreateComment404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	bad := func(msg string) (api.CreateCommentResponseObject, error) {
		return api.CreateComment400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	b := req.Body
	c := model.Comment{ID: model.NewID(), TargetType: string(b.TargetType), TargetID: b.TargetId, AuthorUID: uid, Body: b.Body}
	if b.Mentions != nil {
		c.Mentions = *b.Mentions
	}
	switch c.TargetType {
	case model.TargetStop:
		if h.stops == nil {
			return bad("Comments on stops are not available right now.")
		}
		if _, err := h.liveStop(ctx, req.TripId, c.TargetID); errors.Is(err, repo.ErrNotFound) {
			return bad("targetId is not a stop on this trip.")
		} else if err != nil {
			return nil, err
		}
	case model.TargetTrip:
		if c.TargetID != req.TripId {
			return bad("targetId must be the trip's ID for a trip comment.")
		}
	}
	if err := model.ValidateCommentText(c.Body, c.Mentions); err != nil {
		msg, _ := badRequest(err)
		return bad(msg)
	}
	if msg, err := h.checkMentions(ctx, req.TripId, c.Mentions); err != nil {
		return nil, err
	} else if msg != "" {
		return bad(msg)
	}
	created, err := h.comments.Create(ctx, req.TripId, c)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return bad(msg)
		}
		return nil, err
	}
	return api.CreateComment201JSONResponse(commentFrom(created)), nil
}

// EditComment changes the caller's own comment.
func (h *handlers) EditComment(ctx context.Context, req api.EditCommentRequestObject) (api.EditCommentResponseObject, error) {
	if h.commentsOff() {
		return api.EditComment503JSONResponse{UnavailableJSONResponse: unavailableComments(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "EditComment", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.EditComment404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	cur, err := h.comments.Get(ctx, req.TripId, req.CommentId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.EditComment404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such comment."))}, nil
	}
	if err != nil {
		return nil, err
	}
	if cur.AuthorUID != uid {
		return api.EditComment403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	var mentions []string
	if req.Body.Mentions != nil {
		mentions = *req.Body.Mentions
	}
	bad := func(msg string) (api.EditCommentResponseObject, error) {
		return api.EditComment400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	if err := model.ValidateCommentText(req.Body.Body, mentions); err != nil {
		msg, _ := badRequest(err)
		return bad(msg)
	}
	if msg, err := h.checkMentions(ctx, req.TripId, mentions); err != nil {
		return nil, err
	} else if msg != "" {
		return bad(msg)
	}
	updated, err := h.comments.Edit(ctx, req.TripId, req.CommentId, req.Body.Body, mentions, time.Now())
	if err != nil {
		return nil, err
	}
	return api.EditComment200JSONResponse(commentFrom(updated)), nil
}

// DeleteComment removes a comment. The author or the owner can.
func (h *handlers) DeleteComment(ctx context.Context, req api.DeleteCommentRequestObject) (api.DeleteCommentResponseObject, error) {
	if h.commentsOff() {
		return api.DeleteComment503JSONResponse{UnavailableJSONResponse: unavailableComments(ctx)}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "DeleteComment", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.DeleteComment404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	cur, err := h.comments.Get(ctx, req.TripId, req.CommentId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.DeleteComment204Response{}, nil
	}
	if err != nil {
		return nil, err
	}
	if cur.AuthorUID != uid && !model.Can(role, model.ActionModerate) {
		return api.DeleteComment403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	if err := h.comments.Delete(ctx, req.TripId, req.CommentId); err != nil {
		return nil, err
	}
	return api.DeleteComment204Response{}, nil
}

var _ = slices.Contains[[]string]

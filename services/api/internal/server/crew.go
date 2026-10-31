package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/notify"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithInvites turns on invite links. They also need WithTrips.
func WithInvites(s repo.Invites) Option { return func(h *handlers) { h.invites = s } }

const (
	msgNoCrew        = "The crew is not available right now."
	msgBadInvite     = "This invite link is not valid anymore."
	defaultInviteTTL = 7 * 24 * time.Hour
	fallbackName     = "Traveller"
)

func (h *handlers) invitesOff() bool { return h.trips == nil || h.invites == nil }

// inviteID is the stored ID for a token: its SHA-256 hash, so a leaked
// database does not leak working links.
func inviteID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func inviteFrom(i model.Invite) api.Invite {
	out := api.Invite{
		Id: i.ID, Role: api.InviteRole(i.Role), CreatedBy: i.CreatedBy, CreatedAt: i.CreatedAt,
		UseCount: i.UseCount, Revoked: !i.RevokedAt.IsZero(),
	}
	if !i.ExpiresAt.IsZero() {
		t := i.ExpiresAt
		out.ExpiresAt = &t
	}
	if i.MaxUses > 0 {
		n := i.MaxUses
		out.MaxUses = &n
	}
	return out
}

func memberFrom(m model.Member) api.Member {
	return api.Member{
		Uid: m.UID, Role: api.MemberRole(m.Role), Status: api.MemberStatus(m.Status),
		DisplayName: m.DisplayName, PhotoUrl: strPtr(m.PhotoURL), JoinedAt: m.JoinedAt,
	}
}

// crewRole returns the caller's role, or a deny value for the handler to turn
// into a 404.
func (h *handlers) crewRole(ctx context.Context, op, tripID string) (uid string, role model.Role, deny any, err error) {
	id, err := identity(ctx, op)
	if err != nil {
		return "", "", nil, err
	}
	role, err = h.myMembership(ctx, tripID, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return "", "", api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip.")), nil
	}
	return id.UserID, role, nil, err
}

// ListMembers returns the crew.
func (h *handlers) ListMembers(ctx context.Context, req api.ListMembersRequestObject) (api.ListMembersResponseObject, error) {
	if h.trips == nil {
		return api.ListMembers503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	_, _, deny, err := h.crewRole(ctx, "ListMembers", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListMembers404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	all, err := h.trips.Members(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.MemberList{Members: make([]api.Member, len(all))}
	for i, m := range all {
		out.Members[i] = memberFrom(m)
	}
	return api.ListMembers200JSONResponse(out), nil
}

// SetMemberRole lets the owner switch someone between editor and viewer.
func (h *handlers) SetMemberRole(ctx context.Context, req api.SetMemberRoleRequestObject) (api.SetMemberRoleResponseObject, error) {
	if h.trips == nil {
		return api.SetMemberRole503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	_, role, deny, err := h.crewRole(ctx, "SetMemberRole", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.SetMemberRole404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionManageCrew) {
		return api.SetMemberRole403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	if !req.Body.Role.Valid() {
		return api.SetMemberRole400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "role must be editor or viewer.")}, nil
	}
	err = h.trips.SetMemberRole(ctx, req.TripId, req.Uid, model.Role(req.Body.Role))
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return api.SetMemberRole404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such member."))}, nil
	case errors.Is(err, repo.ErrForbidden):
		return api.SetMemberRole403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, "The owner's role cannot change."))}, nil
	case err != nil:
		return nil, err
	}
	m, err := h.trips.Member(ctx, req.TripId, req.Uid)
	if err != nil {
		return nil, err
	}
	return api.SetMemberRole200JSONResponse(memberFrom(m)), nil
}

// RemoveMember lets the owner remove someone, or anyone leave.
func (h *handlers) RemoveMember(ctx context.Context, req api.RemoveMemberRequestObject) (api.RemoveMemberResponseObject, error) {
	if h.trips == nil {
		return api.RemoveMember503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "RemoveMember", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.RemoveMember404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	status := model.MemberLeft
	if req.Uid != uid {
		if !model.Can(role, model.ActionManageCrew) {
			return api.RemoveMember403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
		}
		status = model.MemberRemoved
		// Only active members can be removed; a missing or past member is a 404.
		if _, err := h.myMembership(ctx, req.TripId, req.Uid); errors.Is(err, repo.ErrNotFound) {
			return api.RemoveMember404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such member."))}, nil
		} else if err != nil {
			return nil, err
		}
	}
	err = h.trips.EndMembership(ctx, req.TripId, req.Uid, status)
	switch {
	case errors.Is(err, repo.ErrForbidden):
		return api.RemoveMember403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, "The owner cannot leave or be removed."))}, nil
	case err != nil:
		return nil, err
	}
	return api.RemoveMember204Response{}, nil
}

// ListInvites returns the trip's invite links without their secrets.
func (h *handlers) ListInvites(ctx context.Context, req api.ListInvitesRequestObject) (api.ListInvitesResponseObject, error) {
	if h.invitesOff() {
		return api.ListInvites503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	_, role, deny, err := h.crewRole(ctx, "ListInvites", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListInvites404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionManageCrew) {
		return api.ListInvites403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	all, err := h.invites.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.InviteList{Invites: make([]api.Invite, len(all))}
	for i, inv := range all {
		out.Invites[i] = inviteFrom(inv)
	}
	return api.ListInvites200JSONResponse(out), nil
}

// CreateInvite makes a link. The token is returned once and only its hash is kept.
func (h *handlers) CreateInvite(ctx context.Context, req api.CreateInviteRequestObject) (api.CreateInviteResponseObject, error) {
	if h.invitesOff() {
		return api.CreateInvite503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "CreateInvite", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.CreateInvite404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionManageCrew) {
		return api.CreateInvite403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	b := req.Body
	ttl, inviteRole, maxUses := defaultInviteTTL, model.RoleEditor, 0
	if b.ExpiresInHours != nil {
		if *b.ExpiresInHours < 1 || *b.ExpiresInHours > 8760 {
			return api.CreateInvite400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "expiresInHours must be between 1 and 8760.")}, nil
		}
		ttl = time.Duration(*b.ExpiresInHours) * time.Hour
	}
	if b.MaxUses != nil {
		if *b.MaxUses < 1 || *b.MaxUses > 1000 {
			return api.CreateInvite400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "maxUses must be between 1 and 1000.")}, nil
		}
		maxUses = *b.MaxUses
	}
	if b.Role != nil {
		if !b.Role.Valid() {
			return api.CreateInvite400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "role must be editor or viewer.")}, nil
		}
		inviteRole = model.Role(*b.Role)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now().UTC().Truncate(time.Microsecond)
	inv := model.Invite{
		ID: inviteID(token), TripID: req.TripId, Role: inviteRole, CreatedBy: uid,
		CreatedAt: now, ExpiresAt: now.Add(ttl), MaxUses: maxUses,
	}
	if err := h.invites.Create(ctx, inv); err != nil {
		return nil, err
	}
	h.award(ctx, uid, model.StampInviteCrew)
	return api.CreateInvite201JSONResponse{Invite: inviteFrom(inv), Token: token}, nil
}

// RevokeInvite ends a link. Owner only.
func (h *handlers) RevokeInvite(ctx context.Context, req api.RevokeInviteRequestObject) (api.RevokeInviteResponseObject, error) {
	if h.invitesOff() {
		return api.RevokeInvite503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	_, role, deny, err := h.crewRole(ctx, "RevokeInvite", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.RevokeInvite404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	notFound := api.RevokeInvite404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such invite."))}
	inv, err := h.invites.Get(ctx, req.InviteId)
	if errors.Is(err, repo.ErrNotFound) || (err == nil && inv.TripID != req.TripId) {
		return notFound, nil
	}
	if err != nil {
		return nil, err
	}
	if !model.Can(role, model.ActionManageCrew) {
		return api.RevokeInvite403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	if err := h.invites.Revoke(ctx, req.TripId, req.InviteId, time.Now()); err != nil {
		return nil, err
	}
	return api.RevokeInvite204Response{}, nil
}

// JoinTrip adds the caller to the trip with the invite's role. It is always free.
func (h *handlers) JoinTrip(ctx context.Context, req api.JoinTripRequestObject) (api.JoinTripResponseObject, error) {
	if h.invitesOff() {
		return api.JoinTrip503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCrew))}, nil
	}
	id, err := identity(ctx, "JoinTrip")
	if err != nil {
		return nil, err
	}
	invalid := api.JoinTrip404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgBadInvite))}
	if req.Body.Token == "" {
		return api.JoinTrip400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "token is required.")}, nil
	}
	now := time.Now()
	inv, err := h.invites.Get(ctx, inviteID(req.Body.Token))
	if errors.Is(err, repo.ErrNotFound) {
		return invalid, nil
	}
	if err != nil {
		return nil, err
	}
	// Someone already on the trip gets it back, even from a spent link, and
	// the invite is not used.
	if role, err := h.myMembership(ctx, inv.TripID, id.UserID); err == nil {
		return h.joined(ctx, inv.TripID, role)
	} else if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	if !inv.Usable(now) {
		return invalid, nil
	}
	if _, err := h.invites.Use(ctx, inv.ID, now); errors.Is(err, repo.ErrNotFound) {
		return invalid, nil
	} else if err != nil {
		return nil, err
	}
	m := model.Member{UID: id.UserID, Role: inv.Role, Status: model.MemberActive, DisplayName: id.DisplayName, InvitedVia: inv.ID, JoinedAt: now.UTC()}
	if h.profile != nil {
		if u, err := h.profile.Get(ctx, id.UserID); err == nil {
			if u.DisplayName != "" {
				m.DisplayName = u.DisplayName
			}
			m.PhotoURL = u.PhotoURL
		} else if !errors.Is(err, repo.ErrNotFound) {
			return nil, err
		}
	}
	if m.DisplayName == "" {
		m.DisplayName = fallbackName
	}
	switch err := h.trips.AddMember(ctx, inv.TripID, m); {
	case err == nil:
		h.emit(ctx, notify.KindMemberJoined, inv.TripID, id.UserID)
	case errors.Is(err, repo.ErrNotFound):
		return invalid, nil // the trip is gone
	case !errors.Is(err, repo.ErrAlreadyExists):
		return nil, err
	}
	return h.joined(ctx, inv.TripID, inv.Role)
}

func (h *handlers) joined(ctx context.Context, tripID string, role model.Role) (api.JoinTripResponseObject, error) {
	t, err := h.trips.Get(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return api.JoinTrip200JSONResponse(tripFrom(t, role)), nil
}

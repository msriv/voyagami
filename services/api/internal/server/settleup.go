package server

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
	"github.com/msriv/voyagami/services/api/internal/settle"
)

// SettlementStore is the payment storage the routes need. repo.Settlements implements it.
type SettlementStore = repo.Settlements

// WithSettleUp turns on balances and payments. They also need WithTrips and WithExpenses.
func WithSettleUp(s SettlementStore) Option { return func(h *handlers) { h.settlements = s } }

const msgNoSettleUp = "Settle-up is not available right now."

func (h *handlers) settleOff() bool {
	return h.trips == nil || h.expenses == nil || h.settlements == nil
}

func settlementFrom(s model.Settlement) api.Settlement {
	out := api.Settlement{
		Id: s.ID, FromUid: s.FromUID, ToUid: s.ToUID, MarkedBy: s.MarkedBy,
		Amount: api.Money{AmountMinor: s.Amount.AmountMinor, Currency: s.Amount.Currency},
	}
	if s.Method != "" {
		out.Method = &s.Method
	}
	return out
}

// balances works out what each person is owed (positive) or owes (negative)
// from the trip's expenses and the payments recorded so far.
func (h *handlers) balances(ctx context.Context, tripID string) (map[string]int64, error) {
	expenses, err := h.expenses.List(ctx, tripID)
	if err != nil {
		return nil, err
	}
	paid, err := h.settlements.List(ctx, tripID)
	if err != nil {
		return nil, err
	}
	spends := make([]settle.Spend, len(expenses))
	for i, e := range expenses {
		spends[i] = settle.Spend{PaidBy: e.PaidByUID, Total: e.Settlement.AmountMinor, Shares: map[string]int64{}}
		for _, sh := range e.Shares {
			spends[i].Shares[sh.UID] += sh.AmountMinor
		}
	}
	pays := make([]settle.Payment, len(paid))
	for i, p := range paid {
		pays[i] = settle.Payment{From: p.FromUID, To: p.ToUID, Amount: p.Amount.AmountMinor}
	}
	bal := settle.Balances(spends, pays)
	return bal, nil
}

// GetBalances returns each member's net balance and the payments that would clear them.
func (h *handlers) GetBalances(ctx context.Context, req api.GetBalancesRequestObject) (api.GetBalancesResponseObject, error) {
	if h.settleOff() {
		return api.GetBalances503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSettleUp))}, nil
	}
	id, err := identity(ctx, "GetBalances")
	if err != nil {
		return nil, err
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.GetBalances404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	bal, err := h.balances(ctx, req.TripId)
	if err != nil {
		return nil, err
	}

	out := api.Balances{Currency: trip.SettlementCurrency, Balances: []api.ExpenseShare{}, Transfers: []api.Transfer{}}
	for uid, a := range bal {
		out.Balances = append(out.Balances, api.ExpenseShare{Uid: uid, AmountMinor: a})
	}
	sort.Slice(out.Balances, func(i, j int) bool { return out.Balances[i].Uid < out.Balances[j].Uid })
	for _, t := range settle.Suggest(bal) {
		out.Transfers = append(out.Transfers, api.Transfer{FromUid: t.From, ToUid: t.To, AmountMinor: t.Amount})
	}
	return api.GetBalances200JSONResponse(out), nil
}

// ListSettlements returns the recorded payments to any member.
func (h *handlers) ListSettlements(ctx context.Context, req api.ListSettlementsRequestObject) (api.ListSettlementsResponseObject, error) {
	if h.settleOff() {
		return api.ListSettlements503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSettleUp))}, nil
	}
	id, err := identity(ctx, "ListSettlements")
	if err != nil {
		return nil, err
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.ListSettlements404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	list, err := h.settlements.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.SettlementList{Settlements: make([]api.Settlement, len(list))}
	for i, s := range list {
		out.Settlements[i] = settlementFrom(s)
	}
	return api.ListSettlements200JSONResponse(out), nil
}

// RecordSettlement marks a payment as paid. The payer, the person paid or the owner can do it.
func (h *handlers) RecordSettlement(ctx context.Context, req api.RecordSettlementRequestObject) (api.RecordSettlementResponseObject, error) {
	if h.settleOff() {
		return api.RecordSettlement503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSettleUp))}, nil
	}
	id, err := identity(ctx, "RecordSettlement")
	if err != nil {
		return nil, err
	}
	role, err := h.myMembership(ctx, req.TripId, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return api.RecordSettlement404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	if err != nil {
		return nil, err
	}
	b := req.Body
	if !model.Can(role, model.ActionModerate) && id.UserID != b.FromUid && id.UserID != b.ToUid {
		return api.RecordSettlement403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	for _, uid := range []string{b.FromUid, b.ToUid} {
		if _, err := h.myMembership(ctx, req.TripId, uid); errors.Is(err, repo.ErrNotFound) {
			return api.RecordSettlement400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Both people must be on the trip.")}, nil
		} else if err != nil {
			return nil, err
		}
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	p := model.Settlement{
		ID: model.NewID(), FromUID: b.FromUid, ToUID: b.ToUid, MarkedBy: id.UserID,
		Amount: model.Money{AmountMinor: b.AmountMinor, Currency: trip.SettlementCurrency},
	}
	if b.Id != nil && *b.Id != "" {
		p.ID = *b.Id
	}
	if b.Method != nil {
		p.Method = strings.TrimSpace(*b.Method)
	}
	saved, err := h.settlements.Create(ctx, req.TripId, p)
	if errors.Is(err, repo.ErrAlreadyExists) {
		old, getErr := h.settlements.Get(ctx, req.TripId, p.ID)
		if getErr == nil && old.MarkedBy == id.UserID {
			return api.RecordSettlement201JSONResponse(settlementFrom(old)), nil
		}
		return api.RecordSettlement400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "That payment ID is already taken.")}, nil
	}
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.RecordSettlement400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.RecordSettlement201JSONResponse(settlementFrom(saved)), nil
}

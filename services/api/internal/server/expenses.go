package server

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/locale"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
	"github.com/msriv/voyagami/services/api/internal/split"
)

// ExpenseStore is the expense storage the routes need. repo.Expenses implements it.
type ExpenseStore = repo.Expenses

// WithExpenses turns on the expense routes. They also need WithTrips.
func WithExpenses(s ExpenseStore) Option { return func(h *handlers) { h.expenses = s } }

const (
	msgNoExpenses  = "Expenses are not available right now."
	msgNeedsPlus   = "Using more than one currency on a trip needs Plus."
	msgNoSuchSpend = "No such trip or expense."
)

func (h *handlers) expensesOff() bool { return h.trips == nil || h.expenses == nil }

func shareList(sh []model.Share) []api.ExpenseShare {
	out := make([]api.ExpenseShare, len(sh))
	for i, s := range sh {
		out[i] = api.ExpenseShare{Uid: s.UID, AmountMinor: s.AmountMinor}
	}
	return out
}

func expenseFrom(e model.Expense) api.Expense {
	out := api.Expense{
		Id: e.ID, Description: e.Description, PaidByUid: e.PaidByUID, Date: e.Date, CreatedBy: e.CreatedBy,
		Original:   api.Money{AmountMinor: e.Original.AmountMinor, Currency: e.Original.Currency},
		Settlement: api.Money{AmountMinor: e.Settlement.AmountMinor, Currency: e.Settlement.Currency},
	}
	if e.Category != "" {
		out.Category = &e.Category
	}
	if e.Rate != nil {
		out.Rate = &api.Rate{Numerator: e.Rate.Numerator, Denominator: e.Rate.Denominator}
	}
	out.Split.Method = api.ExpenseSplitMethod(e.SplitMethod)
	out.Split.Shares = shareList(e.Shares)
	return out
}

// convert returns the original amount in the settlement currency, rounded half away from zero.
func convert(original int64, r model.Rate) (int64, bool) {
	n := new(big.Int).Mul(big.NewInt(original), big.NewInt(r.Numerator))
	den := big.NewInt(r.Denominator)
	neg := n.Sign() < 0
	n.Abs(n)
	n.Mul(n, big.NewInt(2))
	n.Add(n, den)
	n.Div(n, new(big.Int).Mul(den, big.NewInt(2)))
	if neg {
		n.Neg(n)
	}
	return n.Int64(), n.IsInt64()
}

// buildExpense fills in the settlement amount and the shares of e from the
// original amount, the rate and the split. It returns a message for the
// caller when the input is not acceptable.
func (h *handlers) buildExpense(ctx context.Context, trip model.Trip, e model.Expense, in *api.SplitInput) (model.Expense, string, error) {
	if _, err := locale.MinorDigits(e.Original.Currency); err != nil {
		return e, "original currency is not a known ISO 4217 code.", nil
	}
	e.Settlement = model.Money{Currency: trip.SettlementCurrency}
	if e.Original.Currency == trip.SettlementCurrency {
		e.Rate = nil
		e.Settlement.AmountMinor = e.Original.AmountMinor
	} else {
		if e.Rate == nil || e.Rate.Numerator < 1 || e.Rate.Denominator < 1 {
			return e, "rate is needed when the currency differs from the trip's.", nil
		}
		amt, ok := convert(e.Original.AmountMinor, *e.Rate)
		if !ok {
			return e, "amount is too large.", nil
		}
		e.Settlement.AmountMinor = amt
	}

	if in == nil || len(in.Parts) == 0 {
		return e, "split is needed.", nil
	}
	parts := make([]split.Part, len(in.Parts))
	for i, p := range in.Parts {
		parts[i] = split.Part{UID: p.Uid}
		if p.Value != nil {
			parts[i].Value = *p.Value
		}
	}
	got, err := split.Split(e.Settlement.AmountMinor, split.Mode(in.Method), parts)
	if err != nil {
		return e, "split " + err.Error() + ".", nil
	}
	for _, uid := range append([]string{e.PaidByUID}, uidsOf(parts)...) {
		if _, err := h.myMembership(ctx, trip.ID, uid); errors.Is(err, repo.ErrNotFound) {
			return e, "everyone in an expense must be on the trip.", nil
		} else if err != nil {
			return e, "", err
		}
	}
	e.SplitMethod = string(in.Method)
	e.Shares = make([]model.Share, len(got))
	for i, a := range got {
		e.Shares[i] = model.Share{UID: a.UID, AmountMinor: a.AmountMinor}
	}
	return e, "", nil
}

func uidsOf(ps []split.Part) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.UID
	}
	return out
}

// multiCurrencyAllowed asks the entitlements service whether the trip may
// use a second currency. Without the service it is allowed.
func (h *handlers) multiCurrencyAllowed(ctx context.Context, uid, tripID string) (bool, error) {
	if h.plans == nil || h.planSource == nil {
		return true, nil
	}
	user, err := h.planSource.User(ctx, uid)
	if err != nil {
		return false, err
	}
	trip, err := h.planSource.Trip(ctx, tripID)
	if err != nil {
		return false, err
	}
	return h.plans.Check(user, &trip, entitlements.MultiCurrency).Allowed, nil
}

// canChangeExpense says whether the caller may edit or delete e: the owner
// any, an editor only the ones they added or paid for.
func canChangeExpense(role model.Role, uid string, e model.Expense) bool {
	switch role {
	case model.RoleOwner:
		return true
	case model.RoleEditor:
		return e.CreatedBy == uid || e.PaidByUID == uid
	}
	return false
}

// ListExpenses returns a trip's expenses to any member.
func (h *handlers) ListExpenses(ctx context.Context, req api.ListExpensesRequestObject) (api.ListExpensesResponseObject, error) {
	if h.expensesOff() {
		return api.ListExpenses503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExpenses))}, nil
	}
	id, err := identity(ctx, "ListExpenses")
	if err != nil {
		return nil, err
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.ListExpenses404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgNoSuchSpend))}, nil
	} else if err != nil {
		return nil, err
	}
	list, err := h.expenses.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.ExpenseList{Expenses: make([]api.Expense, len(list))}
	for i, e := range list {
		out.Expenses[i] = expenseFrom(e)
	}
	return api.ListExpenses200JSONResponse(out), nil
}

// CreateExpense adds an expense. Owners and editors only.
func (h *handlers) CreateExpense(ctx context.Context, req api.CreateExpenseRequestObject) (api.CreateExpenseResponseObject, error) {
	if h.expensesOff() {
		return api.CreateExpense503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExpenses))}, nil
	}
	id, err := identity(ctx, "CreateExpense")
	if err != nil {
		return nil, err
	}
	role, err := h.myMembership(ctx, req.TripId, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return api.CreateExpense404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgNoSuchSpend))}, nil
	}
	if err != nil {
		return nil, err
	}
	if role == model.RoleViewer {
		return api.CreateExpense403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	b := req.Body
	e := model.Expense{
		ID: model.NewID(), Description: strings.TrimSpace(b.Description), PaidByUID: b.PaidByUid, Date: b.Date,
		Original:  model.Money{AmountMinor: b.Original.AmountMinor, Currency: b.Original.Currency},
		CreatedBy: id.UserID,
	}
	if b.Id != nil && *b.Id != "" {
		e.ID = *b.Id
	}
	if b.Category != nil {
		e.Category = strings.TrimSpace(*b.Category)
	}
	if b.Rate != nil {
		e.Rate = &model.Rate{Numerator: b.Rate.Numerator, Denominator: b.Rate.Denominator}
	}
	if e.Original.Currency != trip.SettlementCurrency {
		if ok, err := h.multiCurrencyAllowed(ctx, id.UserID, req.TripId); err != nil {
			return nil, err
		} else if !ok {
			return api.CreateExpense403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgNeedsPlus))}, nil
		}
	}
	e, msg, err := h.buildExpense(ctx, trip, e, &b.Split)
	if err != nil {
		return nil, err
	}
	if msg != "" {
		return api.CreateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "The expense is not valid: "+msg)}, nil
	}
	saved, err := h.expenses.Create(ctx, req.TripId, e)
	if errors.Is(err, repo.ErrAlreadyExists) {
		// A retry of the same request: return what is stored if this person made it.
		old, getErr := h.expenses.Get(ctx, req.TripId, e.ID)
		if getErr == nil && old.CreatedBy == id.UserID {
			return api.CreateExpense201JSONResponse(expenseFrom(old)), nil
		}
		return api.CreateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "That expense ID is already taken.")}, nil
	}
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.CreateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.CreateExpense201JSONResponse(expenseFrom(saved)), nil
}

// changeAccess loads the expense the caller wants to change and checks they may.
// deny is the response to send, or nil when the change may go ahead.
func (h *handlers) changeAccess(ctx context.Context, op, tripID, expenseID string) (e model.Expense, uid string, deny any, err error) {
	id, err := identity(ctx, op)
	if err != nil {
		return e, "", nil, err
	}
	role, err := h.myMembership(ctx, tripID, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return e, "", api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgNoSuchSpend)), nil
	}
	if err != nil {
		return e, "", nil, err
	}
	e, err = h.expenses.Get(ctx, tripID, expenseID)
	if errors.Is(err, repo.ErrNotFound) {
		return e, "", api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgNoSuchSpend)), nil
	}
	if err != nil {
		return e, "", nil, err
	}
	if !canChangeExpense(role, id.UserID, e) {
		return e, "", errForbidden(ctx), nil
	}
	return e, id.UserID, nil, nil
}

// UpdateExpense changes an expense. The owner can change any; an editor only their own.
func (h *handlers) UpdateExpense(ctx context.Context, req api.UpdateExpenseRequestObject) (api.UpdateExpenseResponseObject, error) {
	if h.expensesOff() {
		return api.UpdateExpense503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExpenses))}, nil
	}
	e, uid, deny, err := h.changeAccess(ctx, "UpdateExpense", req.TripId, req.ExpenseId)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.UpdateExpense404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.UpdateExpense403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	b := req.Body
	if b.Description != nil {
		e.Description = strings.TrimSpace(*b.Description)
	}
	if b.Category != nil {
		e.Category = strings.TrimSpace(*b.Category)
	}
	if b.PaidByUid != nil {
		e.PaidByUID = *b.PaidByUid
	}
	if b.Date != nil {
		e.Date = *b.Date
	}
	money := b.Original != nil || b.Rate != nil
	if b.Original != nil {
		e.Original = model.Money{AmountMinor: b.Original.AmountMinor, Currency: b.Original.Currency}
	}
	if b.Rate != nil {
		e.Rate = &model.Rate{Numerator: b.Rate.Numerator, Denominator: b.Rate.Denominator}
	}
	if money && b.Split == nil {
		return api.UpdateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "The expense is not valid: send split again when the amount or rate changes.")}, nil
	}
	if e.Original.Currency != trip.SettlementCurrency {
		if ok, err := h.multiCurrencyAllowed(ctx, uid, req.TripId); err != nil {
			return nil, err
		} else if !ok {
			return api.UpdateExpense403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgNeedsPlus))}, nil
		}
	}
	if b.Split != nil {
		var msg string
		if e, msg, err = h.buildExpense(ctx, trip, e, b.Split); err != nil {
			return nil, err
		} else if msg != "" {
			return api.UpdateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "The expense is not valid: "+msg)}, nil
		}
	} else if b.PaidByUid != nil {
		if _, err := h.myMembership(ctx, req.TripId, e.PaidByUID); errors.Is(err, repo.ErrNotFound) {
			return api.UpdateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "The expense is not valid: everyone in an expense must be on the trip.")}, nil
		} else if err != nil {
			return nil, err
		}
	}
	saved, err := h.expenses.Replace(ctx, req.TripId, e)
	if errors.Is(err, repo.ErrNotFound) {
		return api.UpdateExpense404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgNoSuchSpend))}, nil
	}
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.UpdateExpense400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.UpdateExpense200JSONResponse(expenseFrom(saved)), nil
}

// DeleteExpense removes an expense from balances. Same role rules as UpdateExpense.
func (h *handlers) DeleteExpense(ctx context.Context, req api.DeleteExpenseRequestObject) (api.DeleteExpenseResponseObject, error) {
	if h.expensesOff() {
		return api.DeleteExpense503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExpenses))}, nil
	}
	_, _, deny, err := h.changeAccess(ctx, "DeleteExpense", req.TripId, req.ExpenseId)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.DeleteExpense404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.DeleteExpense403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	if err := h.expenses.Delete(ctx, req.TripId, req.ExpenseId); err != nil {
		return nil, err
	}
	return api.DeleteExpense204Response{}, nil
}

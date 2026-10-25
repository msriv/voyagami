package server

import (
	"context"
	"errors"
	"sort"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// budgetTarget returns the whole-trip target in the settlement currency.
// ok is false when there is no budget or it is in another currency.
func budgetTarget(t model.Trip, activeMembers int) (target int64, ok bool) {
	b := t.Budget
	if b == nil || b.Amount.Currency != t.SettlementCurrency {
		return 0, false
	}
	if b.Scope == "perPerson" {
		return b.Amount.AmountMinor * int64(activeMembers), true
	}
	return b.Amount.AmountMinor, true
}

// GetBudget sums the trip's expenses against its budget.
func (h *handlers) GetBudget(ctx context.Context, req api.GetBudgetRequestObject) (api.GetBudgetResponseObject, error) {
	if h.expensesOff() {
		return api.GetBudget503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExpenses))}, nil
	}
	id, err := identity(ctx, "GetBudget")
	if err != nil {
		return nil, err
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.GetBudget404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	list, err := h.expenses.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.BudgetSummary{Currency: trip.SettlementCurrency, ByCategory: []struct {
		Category   string `json:"category"`
		SpentMinor int64  `json:"spentMinor"`
	}{}}
	byCat := map[string]int64{}
	for _, e := range list {
		out.SpentMinor += e.Settlement.AmountMinor
		byCat[e.Category] += e.Settlement.AmountMinor
	}
	for c, a := range byCat {
		out.ByCategory = append(out.ByCategory, struct {
			Category   string `json:"category"`
			SpentMinor int64  `json:"spentMinor"`
		}{c, a})
	}
	sort.Slice(out.ByCategory, func(i, j int) bool {
		a, b := out.ByCategory[i], out.ByCategory[j]
		if a.SpentMinor != b.SpentMinor {
			return a.SpentMinor > b.SpentMinor
		}
		return a.Category < b.Category
	})
	if target, ok := budgetTarget(trip, len(trip.MemberUIDs)); ok {
		rem := target - out.SpentMinor
		out.TargetMinor, out.RemainingMinor, out.OverBudget = &target, &rem, rem < 0
	}
	return api.GetBudget200JSONResponse(out), nil
}

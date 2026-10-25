package server

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/locale"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// decimal writes minor units as a decimal using the currency's own minor
// digits: 12345 USD is "123.45", 500 JPY is "500".
func decimal(amount int64, currency string) string {
	digits, err := locale.MinorDigits(currency)
	if err != nil {
		digits = 2
	}
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := strconv.FormatInt(amount, 10)
	if digits > 0 {
		if len(s) <= digits {
			s = strings.Repeat("0", digits-len(s)+1) + s
		}
		s = s[:len(s)-digits] + "." + s[len(s)-digits:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// safeCell stops a spreadsheet from running text that looks like a formula.
func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// ExportExpenses returns the trip's expenses as a CSV file to any member.
func (h *handlers) ExportExpenses(ctx context.Context, req api.ExportExpensesRequestObject) (api.ExportExpensesResponseObject, error) {
	if h.expensesOff() {
		return api.ExportExpenses503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExpenses))}, nil
	}
	id, err := identity(ctx, "ExportExpenses")
	if err != nil {
		return nil, err
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.ExportExpenses404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, msgNoSuchSpend))}, nil
	} else if err != nil {
		return nil, err
	}
	list, err := h.expenses.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}

	names := map[string]string{}
	name := func(uid string) (string, error) {
		if n, ok := names[uid]; ok {
			return n, nil
		}
		m, err := h.trips.Member(ctx, req.TripId, uid)
		n := uid
		if err == nil && m.DisplayName != "" {
			n = m.DisplayName
		} else if err != nil && !errors.Is(err, repo.ErrNotFound) {
			return "", err
		}
		names[uid] = n
		return n, nil
	}

	var buf bytes.Buffer
	buf.WriteString("\uFEFF")
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"date", "description", "category", "paid by", "amount", "currency", "rate", "settlement amount", "settlement currency", "split", "shares"})
	for _, e := range list {
		payer, err := name(e.PaidByUID)
		if err != nil {
			return nil, err
		}
		var shares []string
		for _, sh := range e.Shares {
			n, err := name(sh.UID)
			if err != nil {
				return nil, err
			}
			shares = append(shares, n+": "+decimal(sh.AmountMinor, e.Settlement.Currency))
		}
		rate := ""
		if e.Rate != nil {
			rate = strconv.FormatInt(e.Rate.Numerator, 10) + "/" + strconv.FormatInt(e.Rate.Denominator, 10)
		}
		_ = w.Write([]string{
			e.Date, safeCell(e.Description), safeCell(e.Category), safeCell(payer),
			decimal(e.Original.AmountMinor, e.Original.Currency), e.Original.Currency, rate,
			decimal(e.Settlement.AmountMinor, e.Settlement.Currency), e.Settlement.Currency,
			e.SplitMethod, safeCell(strings.Join(shares, "; ")),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	disp := `attachment; filename="expenses.csv"`
	return api.ExportExpenses200TextcsvResponse{Body: &buf, ContentLength: int64(buf.Len()), Headers: api.ExportExpenses200ResponseHeaders{ContentDisposition: &disp}}, nil
}

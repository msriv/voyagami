package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// fakeExpenses is an in-memory ExpenseStore.
type fakeExpenses struct {
	byID    map[string]model.Expense
	deleted map[string]bool
}

func newFakeExpenses() *fakeExpenses {
	return &fakeExpenses{byID: map[string]model.Expense{}, deleted: map[string]bool{}}
}

func (f *fakeExpenses) Create(_ context.Context, _ string, e model.Expense) (model.Expense, error) {
	if err := e.Validate(); err != nil {
		return model.Expense{}, err
	}
	if _, ok := f.byID[e.ID]; ok {
		return model.Expense{}, repo.ErrAlreadyExists
	}
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeExpenses) Get(_ context.Context, _, id string) (model.Expense, error) {
	e, ok := f.byID[id]
	if !ok || f.deleted[id] {
		return model.Expense{}, repo.ErrNotFound
	}
	return e, nil
}

func (f *fakeExpenses) List(context.Context, string) ([]model.Expense, error) {
	var out []model.Expense
	for id, e := range f.byID {
		if !f.deleted[id] {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeExpenses) Replace(_ context.Context, _ string, e model.Expense) (model.Expense, error) {
	if err := e.Validate(); err != nil {
		return model.Expense{}, err
	}
	old, err := f.Get(context.Background(), "", e.ID)
	if err != nil {
		return model.Expense{}, err
	}
	e.CreatedBy = old.CreatedBy
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeExpenses) Delete(_ context.Context, _, id string) error {
	f.deleted[id] = true
	return nil
}

func TestMoneyConversionRounding(t *testing.T) {
	for _, c := range []struct {
		amount int64
		rate   model.Rate
		want   int64
	}{
		{1000, model.Rate{Numerator: 250, Denominator: 1}, 250000},
		{5, model.Rate{Numerator: 1, Denominator: 2}, 3},
		{-5, model.Rate{Numerator: 1, Denominator: 2}, -3},
		{4, model.Rate{Numerator: 1, Denominator: 3}, 1},
		{1, model.Rate{Numerator: 1, Denominator: 3}, 0},
	} {
		if got, ok := convert(c.amount, c.rate); !ok || got != c.want {
			t.Errorf("convert(%d, %+v) = %d, want %d", c.amount, c.rate, got, c.want)
		}
	}
	if _, ok := convert(1<<62, model.Rate{Numerator: 1 << 20, Denominator: 1}); ok {
		t.Error("an amount that overflows int64 should be refused")
	}
}

func tripBodyIn(currency string) string {
	return strings.Replace(newTripBody, `"settlementCurrency":"VND"`, `"settlementCurrency":"`+currency+`"`, 1)
}

func TestExpenses(t *testing.T) {
	for _, tc := range []struct {
		currency string
		total    int64
		want     []int64
	}{
		// JPY and VND have no minor digits, so these are whole units.
		{"JPY", 1000, []int64{333, 334, 333}},
		{"VND", 250000, []int64{83333, 83334, 83333}},
		{"USD", 1000, []int64{333, 334, 333}},
	} {
		t.Run(tc.currency, func(t *testing.T) {
			trips := newFakeTrips()
			v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
			h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithExpenses(newFakeExpenses()))
			var trip api.Trip
			_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", tripBodyIn(tc.currency)).Body.Bytes(), &trip)
			trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
			trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
			path := "/v1/trips/" + trip.Id + "/expenses"

			body := func(extra string) string {
				return `{"description":"Dinner","paidByUid":"u-owner","date":"2027-02-02","original":{"amountMinor":` +
					strconv.FormatInt(tc.total, 10) + `,"currency":"` + tc.currency + `"},` + extra + `}`
			}
			equal3 := `"split":{"method":"equal","parts":[{"uid":"u-owner"},{"uid":"u-editor"},{"uid":"u-viewer"}]}`

			rec := call(h, "POST", path, "owner", body(equal3))
			if rec.Code != 201 {
				t.Fatalf("create: %d %s", rec.Code, rec.Body)
			}
			var e api.Expense
			_ = json.Unmarshal(rec.Body.Bytes(), &e)
			var got []int64
			for _, s := range e.Split.Shares {
				got = append(got, s.AmountMinor)
			}
			if e.Settlement.AmountMinor != tc.total || e.Rate != nil || len(got) != 3 || got[0] != tc.want[0] || got[1] != tc.want[1] || got[2] != tc.want[2] {
				t.Fatalf("expense: %+v", e)
			}

			// Roles.
			if c := call(h, "POST", path, "viewer", body(equal3)).Code; c != 403 {
				t.Errorf("viewer create: %d", c)
			}
			if c := call(h, "POST", path, "other", body(equal3)).Code; c != 404 {
				t.Errorf("non-member create: %d", c)
			}
			if c := call(h, "GET", path, "other", "").Code; c != 404 {
				t.Errorf("non-member list: %d", c)
			}
			one := path + "/" + e.Id
			if c := call(h, "PATCH", one, "editor", `{"description":"x"}`).Code; c != 403 {
				t.Errorf("editor edits owner's: %d", c)
			}
			if c := call(h, "DELETE", one, "viewer", "").Code; c != 403 {
				t.Errorf("viewer delete: %d", c)
			}

			// Bad input.
			for name, b := range map[string]string{
				"exact short":      body(`"split":{"method":"exact","parts":[{"uid":"u-owner","value":1}]}`),
				"outsider in it":   body(`"split":{"method":"equal","parts":[{"uid":"u-other"}]}`),
				"no split":         body(`"split":{"method":"equal","parts":[]}`),
				"foreign no rate":  strings.Replace(body(equal3), `"currency":"`+tc.currency+`"`, `"currency":"EUR"`, 1),
				"unknown currency": strings.Replace(body(equal3), `"currency":"`+tc.currency+`"`, `"currency":"ZZZ"`, 1),
				"zero":             strings.Replace(body(equal3), `"amountMinor":`+strconv.FormatInt(tc.total, 10), `"amountMinor":0`, 1),
			} {
				if c := call(h, "POST", path, "owner", b).Code; c != 400 {
					t.Errorf("%s: %d", name, c)
				}
			}

			// An editor edits an expense they paid for; amount changes need a new split.
			rec = call(h, "POST", path, "editor", strings.Replace(body(equal3), `"u-owner","date"`, `"u-editor","date"`, 1))
			var mine api.Expense
			_ = json.Unmarshal(rec.Body.Bytes(), &mine)
			if rec.Code != 201 {
				t.Fatalf("editor create: %d %s", rec.Code, rec.Body)
			}
			mineURL := path + "/" + mine.Id
			if c := call(h, "PATCH", mineURL, "editor", `{"original":{"amountMinor":900,"currency":"`+tc.currency+`"}}`).Code; c != 400 {
				t.Errorf("amount without split: %d", c)
			}
			rec = call(h, "PATCH", mineURL, "editor", `{"original":{"amountMinor":900,"currency":"`+tc.currency+`"},`+equal3+`}`)
			var changed api.Expense
			_ = json.Unmarshal(rec.Body.Bytes(), &changed)
			if rec.Code != 200 || changed.Settlement.AmountMinor != 900 || changed.Split.Shares[0].AmountMinor != 300 || changed.CreatedBy != "u-editor" {
				t.Fatalf("edit: %d %s", rec.Code, rec.Body)
			}
			if c := call(h, "PATCH", mineURL, "owner", `{"description":"Lunch"}`).Code; c != 200 {
				t.Errorf("owner edits any: %d", c)
			}

			// Delete leaves the list.
			if c := call(h, "DELETE", one, "owner", "").Code; c != 204 {
				t.Fatalf("delete: %d", c)
			}
			var list api.ExpenseList
			_ = json.Unmarshal(call(h, "GET", path, "viewer", "").Body.Bytes(), &list)
			if len(list.Expenses) != 1 || list.Expenses[0].Id != mine.Id {
				t.Errorf("list after delete: %+v", list)
			}
			if c := call(h, "DELETE", one, "owner", "").Code; c != 404 {
				t.Errorf("delete twice: %d", c)
			}
		})
	}
}

func TestExpenseInOtherCurrencyAndPlusGate(t *testing.T) {
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner"}
	plans := &fakePlans{}
	svc := entitlements.New(entitlements.Free())
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithExpenses(newFakeExpenses()), WithEntitlements(svc, plans))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	path := "/v1/trips/" + trip.Id + "/expenses"
	usd := `{"description":"Taxi","paidByUid":"u-owner","date":"2027-02-02","original":{"amountMinor":1000,"currency":"USD"},` +
		`"rate":{"numerator":250,"denominator":1},"split":{"method":"equal","parts":[{"uid":"u-owner"}]}}`

	if c := call(h, "POST", path, "owner", usd).Code; c != 403 {
		t.Fatalf("free plan, second currency: %d", c)
	}
	plans.trip = entitlements.Trip{Pass: true}
	rec := call(h, "POST", path, "owner", usd)
	var e api.Expense
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != 201 || e.Settlement.AmountMinor != 250000 || e.Settlement.Currency != "VND" || e.Rate == nil || e.Original.Currency != "USD" {
		t.Fatalf("with pass: %d %s", rec.Code, rec.Body)
	}

	// A retry with the same ID returns the stored expense instead of adding another.
	withID := strings.Replace(usd, `{"description"`, `{"id":"e-1","description"`, 1)
	if c := call(h, "POST", path, "owner", withID).Code; c != 201 {
		t.Fatalf("first: %d", c)
	}
	if c := call(h, "POST", path, "owner", withID).Code; c != 201 {
		t.Fatalf("retry: %d", c)
	}
	var list api.ExpenseList
	_ = json.Unmarshal(call(h, "GET", path, "owner", "").Body.Bytes(), &list)
	if len(list.Expenses) != 2 {
		t.Errorf("want 2 expenses, got %d", len(list.Expenses))
	}
}

func TestExpensesOffAnswer503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "GET", "/v1/trips/t/expenses", "owner", "").Code; c != 503 {
		t.Errorf("got %d", c)
	}
}

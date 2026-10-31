// Package export builds the ZIP a person can download of their own data.
package export

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Sources are where the export reads from. A nil source leaves its files out.
type Sources struct {
	Users    repo.Users
	Consents repo.Consents
	Trips    repo.Trips
	Stops    repo.Stops
	Expenses repo.Expenses
	Bookings repo.Bookings
	Journal  repo.Journal
}

// Build returns a ZIP of everything the person has on the app: their profile,
// consent choices, and for every trip they are on the trip, its stops,
// expenses (JSON and CSV), bookings, and the journal entries they wrote.
// Other people's journal entries are not included.
//
// Files use the stored field names. Photos are not included yet: there is no
// photo upload.
func Build(ctx context.Context, s Sources, uid string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, v any) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	if s.Users != nil {
		u, err := s.Users.Get(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("export profile: %w", err)
		}
		if err := add("profile.json", u); err != nil {
			return nil, err
		}
	}
	if s.Consents != nil {
		c, err := s.Consents.Get(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("export consents: %w", err)
		}
		if err := add("consents.json", c); err != nil {
			return nil, err
		}
	}
	if s.Trips != nil {
		trips, err := s.Trips.ListForMember(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("export trips: %w", err)
		}
		if err := add("trips.json", trips); err != nil {
			return nil, err
		}
		for _, t := range trips {
			if err := addTrip(ctx, s, zw, add, t, uid); err != nil {
				return nil, fmt.Errorf("export trip %s: %w", t.ID, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addTrip(ctx context.Context, s Sources, zw *zip.Writer, add func(string, any) error, t model.Trip, uid string) error {
	dir := "trips/" + t.ID + "/"
	if s.Stops != nil {
		stops, err := s.Stops.List(ctx, t.ID)
		if err != nil {
			return err
		}
		if err := add(dir+"stops.json", stops); err != nil {
			return err
		}
	}
	if s.Expenses != nil {
		ex, err := s.Expenses.List(ctx, t.ID)
		if err != nil {
			return err
		}
		if err := add(dir+"expenses.json", ex); err != nil {
			return err
		}
		if err := addExpensesCSV(zw, dir+"expenses.csv", ex); err != nil {
			return err
		}
	}
	if s.Bookings != nil {
		bs, err := s.Bookings.List(ctx, t.ID)
		if err != nil {
			return err
		}
		if err := add(dir+"bookings.json", bs); err != nil {
			return err
		}
	}
	if s.Journal != nil {
		all, err := s.Journal.List(ctx, t.ID)
		if err != nil {
			return err
		}
		mine := []model.JournalEntry{}
		for _, e := range all {
			if e.AuthorUID == uid {
				mine = append(mine, e)
			}
		}
		if err := add(dir+"journal.json", mine); err != nil {
			return err
		}
	}
	return nil
}

// addExpensesCSV writes amounts as integer minor units with their ISO currency
// code, so no locale or float rounding is involved.
func addExpensesCSV(zw *zip.Writer, name string, ex []model.Expense) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"date", "description", "category", "paidBy", "currency", "amountMinor", "settlementCurrency", "settlementAmountMinor", "splitMethod"})
	for _, e := range ex {
		_ = cw.Write([]string{
			e.Date, safeCell(e.Description), safeCell(e.Category), e.PaidByUID, e.Original.Currency, strconv.FormatInt(e.Original.AmountMinor, 10),
			e.Settlement.Currency, strconv.FormatInt(e.Settlement.AmountMinor, 10), e.SplitMethod,
		})
	}
	cw.Flush()
	return cw.Error()
}

// safeCell stops a spreadsheet from running text as a formula.
func safeCell(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}

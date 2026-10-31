package export

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeTrips struct{ repo.Trips }

func (fakeTrips) ListForMember(context.Context, string) ([]model.Trip, error) {
	return []model.Trip{{ID: "t1", Title: "Hanoi"}}, nil
}

type fakeExpenses struct{ repo.Expenses }

func (fakeExpenses) List(context.Context, string) ([]model.Expense, error) {
	return []model.Expense{{
		ID: "e1", Description: "=HYPERLINK(\"x\")", Date: "2026-11-02", PaidByUID: "u1", SplitMethod: "equal",
		Original: model.Money{AmountMinor: 25000, Currency: "VND"}, Settlement: model.Money{AmountMinor: 1, Currency: "USD"},
	}}, nil
}

type fakeJournal struct{ repo.Journal }

func (fakeJournal) List(context.Context, string) ([]model.JournalEntry, error) {
	return []model.JournalEntry{{Date: "2026-11-02", AuthorUID: "u1", Text: "mine"}, {Date: "2026-11-02", AuthorUID: "u2", Text: "theirs"}}, nil
}

func TestBuild(t *testing.T) {
	b, err := Build(context.Background(), Sources{Trips: fakeTrips{}, Expenses: fakeExpenses{}, Journal: fakeJournal{}}, "u1")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		r, _ := f.Open()
		data, _ := io.ReadAll(r)
		_ = r.Close()
		files[f.Name] = string(data)
	}
	for _, name := range []string{"trips.json", "trips/t1/expenses.json", "trips/t1/expenses.csv", "trips/t1/journal.json"} {
		if files[name] == "" {
			t.Errorf("missing %s; have %v", name, keys(files))
		}
	}
	if j := files["trips/t1/journal.json"]; !bytes.Contains([]byte(j), []byte("mine")) || bytes.Contains([]byte(j), []byte("theirs")) {
		t.Errorf("journal must hold only the caller's entries: %s", j)
	}
	rows, err := csv.NewReader(bytes.NewReader([]byte(files["trips/t1/expenses.csv"]))).ReadAll()
	if err != nil || len(rows) != 2 || rows[1][4] != "VND" || rows[1][5] != "25000" || rows[1][1][0] != '\'' {
		t.Errorf("csv %v %v", rows, err)
	}
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

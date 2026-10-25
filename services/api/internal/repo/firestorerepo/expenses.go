package firestorerepo

import (
	"context"
	"errors"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const expensesCollection = "expenses"

// Expenses implements repo.Expenses.
type Expenses struct {
	client *firestore.Client
}

var _ repo.Expenses = (*Expenses)(nil)

// NewExpenses returns the Firestore-backed expense store.
func NewExpenses(client *firestore.Client) *Expenses { return &Expenses{client: client} }

type rateDoc struct {
	Numerator   int64 `firestore:"numerator"`
	Denominator int64 `firestore:"denominator"`
}

type shareDoc struct {
	UID         string `firestore:"uid"`
	AmountMinor int64  `firestore:"amountMinor"`
}

type splitDoc struct {
	Method string     `firestore:"method"`
	Shares []shareDoc `firestore:"shares"`
}

type expenseDoc struct {
	Description string     `firestore:"description"`
	Category    string     `firestore:"category,omitempty"`
	PaidByUID   string     `firestore:"paidByUid"`
	Date        string     `firestore:"date"`
	Original    moneyDoc   `firestore:"original"`
	Rate        *rateDoc   `firestore:"rate,omitempty"`
	Settlement  moneyDoc   `firestore:"settlement"`
	Split       splitDoc   `firestore:"split"`
	Source      string     `firestore:"source"`
	CreatedBy   string     `firestore:"createdBy"`
	CreatedAt   time.Time  `firestore:"createdAt,serverTimestamp"`
	DeletedAt   *time.Time `firestore:"deletedAt,omitempty"`
}

func (s *Expenses) col(tripID string) *firestore.CollectionRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(expensesCollection)
}

func expenseToDoc(e model.Expense) expenseDoc {
	d := expenseDoc{
		Description: e.Description, Category: e.Category, PaidByUID: e.PaidByUID, Date: e.Date,
		Original:   moneyDoc{e.Original.AmountMinor, e.Original.Currency},
		Settlement: moneyDoc{e.Settlement.AmountMinor, e.Settlement.Currency},
		Split:      splitDoc{Method: e.SplitMethod},
		Source:     "typed", CreatedBy: e.CreatedBy,
	}
	if e.Rate != nil {
		d.Rate = &rateDoc{e.Rate.Numerator, e.Rate.Denominator}
	}
	for _, sh := range e.Shares {
		d.Split.Shares = append(d.Split.Shares, shareDoc{sh.UID, sh.AmountMinor})
	}
	return d
}

// expenseFromSnap returns the expense and whether it is deleted.
func expenseFromSnap(snap *firestore.DocumentSnapshot) (model.Expense, bool, error) {
	var d expenseDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Expense{}, false, mapErr("decode expense", err)
	}
	e := model.Expense{
		ID: snap.Ref.ID, Description: d.Description, Category: d.Category, PaidByUID: d.PaidByUID, Date: d.Date,
		Original:    model.Money{AmountMinor: d.Original.AmountMinor, Currency: d.Original.Currency},
		Settlement:  model.Money{AmountMinor: d.Settlement.AmountMinor, Currency: d.Settlement.Currency},
		SplitMethod: d.Split.Method, CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt,
	}
	if d.Rate != nil {
		e.Rate = &model.Rate{Numerator: d.Rate.Numerator, Denominator: d.Rate.Denominator}
	}
	for _, sh := range d.Split.Shares {
		e.Shares = append(e.Shares, model.Share{UID: sh.UID, AmountMinor: sh.AmountMinor})
	}
	return e, d.DeletedAt != nil, nil
}

// Create stores a new expense.
func (s *Expenses) Create(ctx context.Context, tripID string, e model.Expense) (model.Expense, error) {
	if err := e.Validate(); err != nil {
		return model.Expense{}, err
	}
	if _, err := s.col(tripID).Doc(e.ID).Create(ctx, expenseToDoc(e)); err != nil {
		return model.Expense{}, mapErr("create expense", err)
	}
	return s.Get(ctx, tripID, e.ID)
}

// Get returns a live expense.
func (s *Expenses) Get(ctx context.Context, tripID, expenseID string) (model.Expense, error) {
	snap, err := s.col(tripID).Doc(expenseID).Get(ctx)
	if err != nil {
		return model.Expense{}, mapErr("get expense", err)
	}
	e, deleted, err := expenseFromSnap(snap)
	if err != nil {
		return model.Expense{}, err
	}
	if deleted {
		return model.Expense{}, repo.ErrNotFound
	}
	return e, nil
}

// List returns live expenses, newest date first, then newest created first.
func (s *Expenses) List(ctx context.Context, tripID string) ([]model.Expense, error) {
	iter := s.col(tripID).Documents(ctx)
	defer iter.Stop()
	var out []model.Expense
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list expenses", err)
		}
		e, deleted, err := expenseFromSnap(snap)
		if err != nil {
			return nil, err
		}
		if !deleted {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Date != b.Date {
			return a.Date > b.Date
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID < b.ID
	})
	return out, nil
}

// Replace overwrites the content of a live expense in one transaction.
func (s *Expenses) Replace(ctx context.Context, tripID string, e model.Expense) (model.Expense, error) {
	if err := e.Validate(); err != nil {
		return model.Expense{}, err
	}
	ref := s.col(tripID).Doc(e.ID)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		_, deleted, err := expenseFromSnap(snap)
		if err != nil {
			return err
		}
		if deleted {
			return repo.ErrNotFound
		}
		d := expenseToDoc(e)
		ups := []firestore.Update{
			{Path: "description", Value: d.Description},
			{Path: "paidByUid", Value: d.PaidByUID},
			{Path: "date", Value: d.Date},
			{Path: "original", Value: d.Original},
			{Path: "settlement", Value: d.Settlement},
			{Path: "split", Value: d.Split},
			{Path: "category", Value: firestore.Delete},
			{Path: "rate", Value: firestore.Delete},
		}
		if d.Category != "" {
			ups[6].Value = d.Category
		}
		if d.Rate != nil {
			ups[7].Value = d.Rate
		}
		return tx.Update(ref, ups)
	})
	if err != nil {
		return model.Expense{}, mapErr("replace expense", err)
	}
	return s.Get(ctx, tripID, e.ID)
}

// Delete marks an expense deleted.
func (s *Expenses) Delete(ctx context.Context, tripID, expenseID string) error {
	_, err := s.col(tripID).Doc(expenseID).Update(ctx, []firestore.Update{{Path: "deletedAt", Value: firestore.ServerTimestamp}})
	if isNotFound(err) {
		return nil
	}
	return mapErr("delete expense", err)
}

package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const supportCollection = "supportMessages"

// Support implements repo.Support on the supportMessages collection.
type Support struct {
	client *firestore.Client
}

var _ repo.Support = (*Support)(nil)

// NewSupport returns the Firestore-backed support inbox.
func NewSupport(client *firestore.Client) *Support { return &Support{client: client} }

type supportDoc struct {
	UserID      string    `firestore:"uid"`
	Category    string    `firestore:"category"`
	Text        string    `firestore:"text"`
	CreatedAt   time.Time `firestore:"createdAt"`
	Diagnostics struct {
		AppVersion  string `firestore:"appVersion"`
		OS          string `firestore:"os"`
		OSVersion   string `firestore:"osVersion"`
		DeviceModel string `firestore:"deviceModel"`
		Locale      string `firestore:"locale"`
		RequestID   string `firestore:"requestId"`
	} `firestore:"diagnostics"`
}

// Add stores a message.
func (s *Support) Add(ctx context.Context, m model.SupportMessage) (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	d := supportDoc{UserID: m.UserID, Category: m.Category, Text: m.Text, CreatedAt: m.CreatedAt.UTC().Truncate(time.Microsecond)}
	d.Diagnostics.AppVersion, d.Diagnostics.OS, d.Diagnostics.OSVersion = m.Diagnostics.AppVersion, m.Diagnostics.OS, m.Diagnostics.OSVersion
	d.Diagnostics.DeviceModel, d.Diagnostics.Locale, d.Diagnostics.RequestID = m.Diagnostics.DeviceModel, m.Diagnostics.Locale, m.Diagnostics.RequestID
	id := model.NewID()
	_, err := s.client.Collection(supportCollection).Doc(id).Create(ctx, d)
	return id, mapErr("add support message", err)
}

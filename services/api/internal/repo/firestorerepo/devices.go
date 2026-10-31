package firestorerepo

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const devicesCollection = "devices"

// Devices implements repo.Devices under users/{uid}/devices.
type Devices struct {
	client *firestore.Client
}

var _ repo.Devices = (*Devices)(nil)

// NewDevices returns the Firestore-backed device store.
func NewDevices(client *firestore.Client) *Devices { return &Devices{client: client} }

type deviceDoc struct {
	Platform   string    `firestore:"platform"`
	PushToken  string    `firestore:"pushToken"`
	LastSeenAt time.Time `firestore:"lastSeenAt"`
}

func (s *Devices) coll(uid string) *firestore.CollectionRef {
	return s.client.Collection(usersCollection).Doc(uid).Collection(devicesCollection)
}

// Put stores or refreshes a device.
func (s *Devices) Put(ctx context.Context, uid string, d model.Device) error {
	if err := d.Validate(); err != nil {
		return err
	}
	_, err := s.coll(uid).Doc(d.ID).Set(ctx, deviceDoc{string(d.Platform), d.PushToken, d.LastSeenAt.UTC().Truncate(time.Microsecond)})
	return mapErr("put device", err)
}

// Delete removes a device; a missing one is fine.
func (s *Devices) Delete(ctx context.Context, uid, deviceID string) error {
	if deviceID == "" {
		return nil
	}
	_, err := s.coll(uid).Doc(deviceID).Delete(ctx)
	return mapErr("delete device", err)
}

// List returns the person's devices.
func (s *Devices) List(ctx context.Context, uid string) ([]model.Device, error) {
	iter := s.coll(uid).Documents(ctx)
	defer iter.Stop()
	var out []model.Device
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list devices", err)
		}
		var d deviceDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode device", err)
		}
		out = append(out, model.Device{ID: snap.Ref.ID, Platform: model.Platform(d.Platform), PushToken: d.PushToken, LastSeenAt: d.LastSeenAt})
	}
	return out, nil
}

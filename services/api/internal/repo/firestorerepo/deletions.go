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

const deletionsCollection = "deletionRequests"

// Deletions implements repo.Deletions on deletionRequests/{uid}.
type Deletions struct {
	client *firestore.Client
}

var _ repo.Deletions = (*Deletions)(nil)

// NewDeletions returns the Firestore-backed deletion store.
func NewDeletions(client *firestore.Client) *Deletions { return &Deletions{client: client} }

type deletionDoc struct {
	RequestedAt time.Time  `firestore:"requestedAt"`
	DeleteOn    time.Time  `firestore:"deleteOn"`
	PurgedAt    *time.Time `firestore:"purgedAt,omitempty"`
}

func (s *Deletions) ref(uid string) *firestore.DocumentRef {
	return s.client.Collection(deletionsCollection).Doc(uid)
}

// Schedule records the request, keeping the first date if one exists.
func (s *Deletions) Schedule(ctx context.Context, uid string, now time.Time) (model.Deletion, error) {
	var out model.Deletion
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(s.ref(uid))
		if err == nil {
			var d deletionDoc
			if err := snap.DataTo(&d); err != nil {
				return err
			}
			if d.PurgedAt == nil {
				out = model.Deletion{UserID: uid, RequestedAt: d.RequestedAt, DeleteOn: d.DeleteOn}
				return nil
			}
		} else if !isNotFound(err) {
			return err
		}
		now = now.UTC().Truncate(time.Microsecond)
		out = model.Deletion{UserID: uid, RequestedAt: now, DeleteOn: now.Add(model.DeletionGrace)}
		return tx.Set(s.ref(uid), deletionDoc{RequestedAt: out.RequestedAt, DeleteOn: out.DeleteOn})
	})
	return out, mapErr("schedule deletion", err)
}

// Get returns the pending request, or ErrNotFound.
func (s *Deletions) Get(ctx context.Context, uid string) (model.Deletion, error) {
	snap, err := s.ref(uid).Get(ctx)
	if err != nil {
		return model.Deletion{}, mapErr("get deletion", err)
	}
	var d deletionDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Deletion{}, mapErr("decode deletion", err)
	}
	if d.PurgedAt != nil {
		return model.Deletion{}, repo.ErrNotFound
	}
	return model.Deletion{UserID: uid, RequestedAt: d.RequestedAt, DeleteOn: d.DeleteOn}, nil
}

// Cancel drops a pending request. A request that was already carried out stays.
func (s *Deletions) Cancel(ctx context.Context, uid string) error {
	return mapErr("cancel deletion", s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(s.ref(uid))
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		var d deletionDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.PurgedAt != nil {
			return nil
		}
		return tx.Delete(s.ref(uid))
	}))
}

// Due returns the people whose date has come and who are not purged yet.
func (s *Deletions) Due(ctx context.Context, now time.Time, limit int) ([]string, error) {
	iter := s.client.Collection(deletionsCollection).Where("deleteOn", "<=", now).OrderBy("deleteOn", firestore.Asc).Limit(limit).Documents(ctx)
	defer iter.Stop()
	var out []string
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, mapErr("list due deletions", err)
		}
		var d deletionDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode deletion", err)
		}
		if d.PurgedAt == nil {
			out = append(out, snap.Ref.ID)
		}
	}
}

// Purge deletes the person's data. Every step can be repeated, so a run that
// stopped halfway is finished by the next one.
func (s *Deletions) Purge(ctx context.Context, uid string, now time.Time) error {
	bw := s.client.BulkWriter(ctx)
	if err := s.leaveTrips(ctx, bw, uid); err != nil {
		bw.End()
		return err
	}
	for _, ref := range []*firestore.DocumentRef{
		s.client.Collection(usersCollection).Doc(uid),
		s.client.Collection(consentsCollection).Doc(uid),
		s.client.Collection(entitlementsCollection).Doc(uid),
	} {
		if err := s.deleteTree(ctx, bw, ref); err != nil {
			bw.End()
			return err
		}
	}
	for _, q := range []firestore.Query{
		s.client.Collection(sessionsCollection).Where("userId", "==", uid),
		s.client.Collection(phoneIndexCollection).Where("uid", "==", uid),
	} {
		if err := s.deleteQuery(ctx, bw, q); err != nil {
			bw.End()
			return err
		}
	}
	bw.End()
	_, err := s.ref(uid).Set(ctx, map[string]any{"purgedAt": now.UTC()}, firestore.MergeAll)
	return mapErr("finish deletion", err)
}

// leaveTrips takes the person out of every trip they are on. A trip nobody
// else is on goes away with them. On a shared trip they stay as a member who
// left, with no name, photo or payment details, so expenses and comments keep
// making sense; their journal entries and live location go. If they owned it,
// the member who joined first becomes the owner.
func (s *Deletions) leaveTrips(ctx context.Context, bw *firestore.BulkWriter, uid string) error {
	iter := s.client.Collection(tripsCollection).Where("memberUids", "array-contains", uid).Documents(ctx)
	defer iter.Stop()
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return mapErr("list trips to leave", err)
		}
		if err := s.leaveTrip(ctx, bw, snap.Ref, uid); err != nil {
			return err
		}
	}
}

func (s *Deletions) leaveTrip(ctx context.Context, bw *firestore.BulkWriter, trip *firestore.DocumentRef, uid string) error {
	members, err := trip.Collection(membersCollection).Documents(ctx).GetAll()
	if err != nil {
		return mapErr("list members", err)
	}
	var heir *firestore.DocumentRef
	var heirJoined time.Time
	for _, m := range members {
		var d memberDoc
		if err := m.DataTo(&d); err != nil {
			return mapErr("decode member", err)
		}
		if m.Ref.ID == uid || d.Status != string(model.MemberActive) {
			continue
		}
		if heir == nil || d.JoinedAt.Before(heirJoined) {
			heir, heirJoined = m.Ref, d.JoinedAt
		}
	}
	if heir == nil {
		return s.deleteTree(ctx, bw, trip)
	}
	if err := s.deleteQuery(ctx, bw, trip.Collection(journalCollection).Where("authorUid", "==", uid)); err != nil {
		return err
	}
	_, _ = bw.Delete(trip.Collection(presenceCollection).Doc(uid))
	return mapErr("leave trip", s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		ts, err := tx.Get(trip)
		if err != nil {
			return err
		}
		var t struct {
			OwnerUID string `firestore:"ownerUid"`
		}
		if err := ts.DataTo(&t); err != nil {
			return err
		}
		me := trip.Collection(membersCollection).Doc(uid)
		if err := tx.Update(me, []firestore.Update{
			{Path: "status", Value: string(model.MemberLeft)}, {Path: "displayName", Value: ""},
			{Path: "photoUrl", Value: firestore.Delete}, {Path: "paymentHandles", Value: []paymentHandleDoc{}},
		}); err != nil {
			return err
		}
		ups := []firestore.Update{{Path: "memberUids", Value: firestore.ArrayRemove(uid)}, {Path: "updatedAt", Value: firestore.ServerTimestamp}}
		if t.OwnerUID == uid {
			ups = append(ups, firestore.Update{Path: "ownerUid", Value: heir.ID})
			if err := tx.Update(heir, []firestore.Update{{Path: "role", Value: string(model.RoleOwner)}}); err != nil {
				return err
			}
		}
		return tx.Update(trip, ups)
	}))
}

func (s *Deletions) deleteQuery(ctx context.Context, bw *firestore.BulkWriter, q firestore.Query) error {
	refs, err := q.Documents(ctx).GetAll()
	if err != nil {
		return mapErr("list documents to delete", err)
	}
	for _, r := range refs {
		if _, err := bw.Delete(r.Ref); err != nil {
			return mapErr("delete document", err)
		}
	}
	return nil
}

// deleteTree deletes a document and everything under it.
func (s *Deletions) deleteTree(ctx context.Context, bw *firestore.BulkWriter, ref *firestore.DocumentRef) error {
	colls := ref.Collections(ctx)
	for {
		c, err := colls.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return mapErr("list collections", err)
		}
		docs := c.DocumentRefs(ctx)
		for {
			d, err := docs.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return mapErr("list documents", err)
			}
			if err := s.deleteTree(ctx, bw, d); err != nil {
				return err
			}
		}
	}
	if _, err := bw.Delete(ref); err != nil {
		return mapErr("delete document", err)
	}
	return nil
}

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

const recapVideosCollection = "recapVideos"

// RecapVideos implements repo.RecapVideos.
type RecapVideos struct{ client *firestore.Client }

var _ repo.RecapVideos = (*RecapVideos)(nil)

// NewRecapVideos returns the Firestore-backed render queue.
func NewRecapVideos(client *firestore.Client) *RecapVideos { return &RecapVideos{client: client} }

type recapVideoDoc struct {
	TripID      string    `firestore:"tripId"`
	RequestedBy string    `firestore:"requestedBy"`
	State       string    `firestore:"state"`
	Progress    int       `firestore:"progress"`
	MaxSeconds  int       `firestore:"maxSeconds"`
	Watermark   bool      `firestore:"watermark"`
	Object      string    `firestore:"object"`
	Error       string    `firestore:"error"`
	Attempts    int       `firestore:"attempts"`
	CreatedAt   time.Time `firestore:"createdAt"`
	StartedAt   time.Time `firestore:"startedAt"`
}

func (d recapVideoDoc) video(id string) model.RecapVideo {
	return model.RecapVideo{ID: id, TripID: d.TripID, RequestedBy: d.RequestedBy, State: model.RecapVideoState(d.State), Progress: d.Progress,
		MaxSeconds: d.MaxSeconds, Watermark: d.Watermark, Object: d.Object, Error: d.Error, Attempts: d.Attempts, CreatedAt: d.CreatedAt, StartedAt: d.StartedAt}
}

func (s *RecapVideos) coll() *firestore.CollectionRef {
	return s.client.Collection(recapVideosCollection)
}

// Enqueue stores the job unless the trip already has an active one.
func (s *RecapVideos) Enqueue(ctx context.Context, v model.RecapVideo) (model.RecapVideo, bool, error) {
	var out model.RecapVideo
	created := false
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		created = false
		it := tx.Documents(s.coll().Where("tripId", "==", v.TripID))
		defer it.Stop()
		for {
			snap, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			var d recapVideoDoc
			if err := snap.DataTo(&d); err != nil {
				return err
			}
			if j := d.video(snap.Ref.ID); j.Active() {
				out = j
				return nil
			}
		}
		ref := s.coll().NewDoc()
		d := recapVideoDoc{TripID: v.TripID, RequestedBy: v.RequestedBy, State: string(model.RecapQueued), MaxSeconds: v.MaxSeconds,
			Watermark: v.Watermark, CreatedAt: v.CreatedAt.UTC()}
		out, created = d.video(ref.ID), true
		return tx.Create(ref, d)
	})
	return out, created, mapErr("enqueue recap video", err)
}

// Latest returns the trip's newest job. A trip has few jobs, so it picks in memory.
func (s *RecapVideos) Latest(ctx context.Context, tripID string) (model.RecapVideo, error) {
	docs, err := s.coll().Where("tripId", "==", tripID).Documents(ctx).GetAll()
	if err != nil {
		return model.RecapVideo{}, mapErr("get recap video", err)
	}
	var best model.RecapVideo
	found := false
	for _, snap := range docs {
		var d recapVideoDoc
		if err := snap.DataTo(&d); err != nil {
			return model.RecapVideo{}, mapErr("decode recap video", err)
		}
		if v := d.video(snap.Ref.ID); !found || v.CreatedAt.After(best.CreatedAt) {
			best, found = v, true
		}
	}
	if !found {
		return model.RecapVideo{}, repo.ErrNotFound
	}
	return best, nil
}

// Claim takes the oldest queued job, or one stuck rendering, in a transaction.
func (s *RecapVideos) Claim(ctx context.Context, now, staleBefore time.Time) (model.RecapVideo, bool, error) {
	var out model.RecapVideo
	ok := false
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		ok = false
		var pick *firestore.DocumentSnapshot
		var pickAt time.Time
		for _, state := range []model.RecapVideoState{model.RecapQueued, model.RecapRendering} {
			snaps, err := tx.Documents(s.coll().Where("state", "==", string(state)).Limit(20)).GetAll()
			if err != nil {
				return err
			}
			for _, snap := range snaps {
				var d recapVideoDoc
				if err := snap.DataTo(&d); err != nil {
					return err
				}
				if state == model.RecapRendering && !d.StartedAt.Before(staleBefore) {
					continue
				}
				if pick == nil || d.CreatedAt.Before(pickAt) {
					pick, pickAt = snap, d.CreatedAt
				}
			}
		}
		if pick == nil {
			return nil
		}
		var d recapVideoDoc
		if err := pick.DataTo(&d); err != nil {
			return err
		}
		d.State, d.StartedAt, d.Attempts, d.Progress = string(model.RecapRendering), now.UTC(), d.Attempts+1, 0
		out, ok = d.video(pick.Ref.ID), true
		return tx.Set(pick.Ref, d)
	})
	return out, ok, mapErr("claim recap video", err)
}

func (s *RecapVideos) update(ctx context.Context, id string, u []firestore.Update) error {
	_, err := s.coll().Doc(id).Update(ctx, u)
	return mapErr("update recap video", err)
}

// Progress records how far the job is.
func (s *RecapVideos) Progress(ctx context.Context, id string, percent int) error {
	return s.update(ctx, id, []firestore.Update{{Path: "progress", Value: percent}})
}

// Finish marks the job ready.
func (s *RecapVideos) Finish(ctx context.Context, id, object string) error {
	return s.update(ctx, id, []firestore.Update{{Path: "state", Value: string(model.RecapReady)}, {Path: "progress", Value: 100}, {Path: "object", Value: object}, {Path: "error", Value: ""}})
}

// Fail ends an attempt.
func (s *RecapVideos) Fail(ctx context.Context, id, message string, requeue bool) error {
	if requeue {
		return s.update(ctx, id, []firestore.Update{{Path: "state", Value: string(model.RecapQueued)}})
	}
	return s.update(ctx, id, []firestore.Update{{Path: "state", Value: string(model.RecapFailed)}, {Path: "error", Value: message}})
}

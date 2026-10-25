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

const tasksCollection = "tasks"

// Tasks implements repo.Tasks.
type Tasks struct {
	client *firestore.Client
}

var _ repo.Tasks = (*Tasks)(nil)

// NewTasks returns the Firestore-backed task store.
func NewTasks(client *firestore.Client) *Tasks { return &Tasks{client: client} }

type taskDoc struct {
	Title     string     `firestore:"title"`
	Kind      string     `firestore:"kind"`
	Done      bool       `firestore:"done"`
	DoneBy    string     `firestore:"doneBy,omitempty"`
	DoneAt    *time.Time `firestore:"doneAt,omitempty"`
	CreatedAt time.Time  `firestore:"createdAt,serverTimestamp"`
}

func (s *Tasks) ref(tripID, taskID string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(tasksCollection).Doc(taskID)
}

func taskFromSnap(snap *firestore.DocumentSnapshot) (model.Task, error) {
	var d taskDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Task{}, mapErr("decode task", err)
	}
	t := model.Task{ID: snap.Ref.ID, Title: d.Title, Done: d.Done, DoneBy: d.DoneBy, CreatedAt: d.CreatedAt}
	if d.DoneAt != nil {
		t.DoneAt = *d.DoneAt
	}
	return t, nil
}

// Create stores a custom task.
func (s *Tasks) Create(ctx context.Context, tripID string, t model.Task) (model.Task, error) {
	if err := t.Validate(); err != nil {
		return model.Task{}, err
	}
	if _, err := s.ref(tripID, t.ID).Create(ctx, taskDoc{Title: t.Title, Kind: "custom"}); err != nil {
		return model.Task{}, mapErr("create task", err)
	}
	snap, err := s.ref(tripID, t.ID).Get(ctx)
	if err != nil {
		return model.Task{}, mapErr("get task", err)
	}
	return taskFromSnap(snap)
}

// List returns open tasks first, then done ones, oldest first within each.
func (s *Tasks) List(ctx context.Context, tripID string) ([]model.Task, error) {
	iter := s.client.Collection(tripsCollection).Doc(tripID).Collection(tasksCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.Task
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list tasks", err)
		}
		t, err := taskFromSnap(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Done != b.Done {
			return !a.Done
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.ID < b.ID
	})
	return out, nil
}

// Update changes only the fields set in the patch.
func (s *Tasks) Update(ctx context.Context, tripID, taskID string, p model.TaskPatch) (model.Task, error) {
	if err := p.Validate(); err != nil {
		return model.Task{}, err
	}
	var ups []firestore.Update
	if p.Title != nil {
		ups = append(ups, firestore.Update{Path: "title", Value: *p.Title})
	}
	if p.Done != nil {
		ups = append(ups, firestore.Update{Path: "done", Value: *p.Done})
		if *p.Done {
			ups = append(ups,
				firestore.Update{Path: "doneBy", Value: p.By},
				firestore.Update{Path: "doneAt", Value: firestore.ServerTimestamp})
		} else {
			ups = append(ups,
				firestore.Update{Path: "doneBy", Value: firestore.Delete},
				firestore.Update{Path: "doneAt", Value: firestore.Delete})
		}
	}
	if len(ups) > 0 {
		if _, err := s.ref(tripID, taskID).Update(ctx, ups); err != nil {
			return model.Task{}, mapErr("update task", err)
		}
	}
	snap, err := s.ref(tripID, taskID).Get(ctx)
	if err != nil {
		return model.Task{}, mapErr("get task", err)
	}
	return taskFromSnap(snap)
}

// Delete removes a task.
func (s *Tasks) Delete(ctx context.Context, tripID, taskID string) error {
	_, err := s.ref(tripID, taskID).Delete(ctx)
	return mapErr("delete task", err)
}

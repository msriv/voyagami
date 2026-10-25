package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// fakeTasks is an in-memory TaskStore with the same validation as the real one.
type fakeTasks struct {
	tasks map[string]map[string]model.Task
	n     int
}

func newFakeTasks() *fakeTasks { return &fakeTasks{tasks: map[string]map[string]model.Task{}} }

func (f *fakeTasks) Create(_ context.Context, tripID string, t model.Task) (model.Task, error) {
	if err := t.Validate(); err != nil {
		return model.Task{}, err
	}
	if f.tasks[tripID] == nil {
		f.tasks[tripID] = map[string]model.Task{}
	}
	f.n++
	f.tasks[tripID][t.ID] = t
	return t, nil
}

func (f *fakeTasks) List(_ context.Context, tripID string) ([]model.Task, error) {
	var out []model.Task
	for _, t := range f.tasks[tripID] {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

func (f *fakeTasks) Update(_ context.Context, tripID, id string, p model.TaskPatch) (model.Task, error) {
	if err := p.Validate(); err != nil {
		return model.Task{}, err
	}
	t, ok := f.tasks[tripID][id]
	if !ok {
		return model.Task{}, repo.ErrNotFound
	}
	if p.Title != nil {
		t.Title = *p.Title
	}
	if p.Done != nil {
		t.Done = *p.Done
	}
	f.tasks[tripID][id] = t
	return t, nil
}

func (f *fakeTasks) Delete(_ context.Context, tripID, id string) error {
	delete(f.tasks[tripID], id)
	return nil
}

func TestTasksAccessAndLifecycle(t *testing.T) {
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "other": "u-other", "viewer": "u-viewer", "editor": "u-editor"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithTasks(newFakeTasks()))

	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	base := "/v1/trips/" + trip.Id + "/tasks"

	rec := call(h, "POST", base, "editor", `{"title":"  Book SIM  "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var task api.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &task)
	if task.Title != "Book SIM" || task.Done {
		t.Fatalf("task = %+v", task)
	}
	one := base + "/" + task.Id

	tests := []struct {
		name, method, path, token, body string
		want                            int
	}{
		{"viewer reads", "GET", base, "viewer", "", 200},
		{"non-member cannot read", "GET", base, "other", "", 404},
		{"signed out", "GET", base, "", "", 401},
		{"viewer cannot add", "POST", base, "viewer", `{"title":"x"}`, 403},
		{"non-member cannot add", "POST", base, "other", `{"title":"x"}`, 404},
		{"blank title", "POST", base, "owner", `{"title":"  "}`, 400},
		{"long title", "POST", base, "owner", `{"title":"` + `1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"}`, 400},
		{"editor ticks", "PATCH", one, "editor", `{"done":true}`, 200},
		{"viewer cannot tick", "PATCH", one, "viewer", `{"done":false}`, 403},
		{"missing task", "PATCH", base + "/nope", "owner", `{"done":true}`, 404},
		{"viewer cannot delete", "DELETE", one, "viewer", "", 403},
		{"owner deletes", "DELETE", one, "owner", "", 204},
		{"delete again", "DELETE", one, "owner", "", 204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := call(h, tt.method, tt.path, tt.token, tt.body).Code; got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
	var list api.TaskList
	_ = json.Unmarshal(call(h, "GET", base, "owner", "").Body.Bytes(), &list)
	if len(list.Tasks) != 0 {
		t.Fatalf("list after delete = %+v", list)
	}
}

func TestTasksUnavailableWithoutStore(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u"}, WithTrips(newFakeTrips()))
	if got := call(h, "GET", "/v1/trips/x/tasks", "owner", "").Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}

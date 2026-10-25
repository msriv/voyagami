package server

import (
	"context"
	"errors"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// TaskStore is the task storage the routes need. repo.Tasks implements it.
type TaskStore = repo.Tasks

// WithTasks turns on the task routes. They also need WithTrips, which says who is on a trip.
func WithTasks(s TaskStore) Option { return func(h *handlers) { h.tasks = s } }

const msgNoTasks = "To-dos are not available right now."

func (h *handlers) tasksReady() bool { return h.trips != nil && h.tasks != nil }

func taskFrom(t model.Task) api.Task { return api.Task{Id: t.ID, Title: t.Title, Done: t.Done} }

// taskAccess finds the caller's role on the trip. ok is false when they are not a member.
func (h *handlers) taskAccess(ctx context.Context, op, tripID string) (role model.Role, uid string, ok bool, err error) {
	id, err := identity(ctx, op)
	if err != nil {
		return "", "", false, err
	}
	role, err = h.myMembership(ctx, tripID, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return "", "", false, nil
	}
	return role, id.UserID, err == nil, err
}

func (h *handlers) canEditTasks(role model.Role) bool { return role != model.RoleViewer }

// ListTasks returns a trip's to-dos to any member.
func (h *handlers) ListTasks(ctx context.Context, req api.ListTasksRequestObject) (api.ListTasksResponseObject, error) {
	if !h.tasksReady() {
		return api.ListTasks503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTasks))}, nil
	}
	_, _, ok, err := h.taskAccess(ctx, "ListTasks", req.TripId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return api.ListTasks404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	tasks, err := h.tasks.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.TaskList{Tasks: make([]api.Task, len(tasks))}
	for i, t := range tasks {
		out.Tasks[i] = taskFrom(t)
	}
	return api.ListTasks200JSONResponse(out), nil
}

// CreateTask adds a to-do. Owners and editors only.
func (h *handlers) CreateTask(ctx context.Context, req api.CreateTaskRequestObject) (api.CreateTaskResponseObject, error) {
	if !h.tasksReady() {
		return api.CreateTask503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTasks))}, nil
	}
	role, _, ok, err := h.taskAccess(ctx, "CreateTask", req.TripId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return api.CreateTask404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	if !h.canEditTasks(role) {
		return api.CreateTask403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgRoleDenied))}, nil
	}
	t, err := h.tasks.Create(ctx, req.TripId, model.Task{ID: model.NewID(), Title: strings.TrimSpace(req.Body.Title)})
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.CreateTask400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, msg))}, nil
		}
		return nil, err
	}
	return api.CreateTask201JSONResponse(taskFrom(t)), nil
}

// UpdateTask ticks, unticks or renames a to-do. Owners and editors only.
func (h *handlers) UpdateTask(ctx context.Context, req api.UpdateTaskRequestObject) (api.UpdateTaskResponseObject, error) {
	if !h.tasksReady() {
		return api.UpdateTask503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTasks))}, nil
	}
	role, uid, ok, err := h.taskAccess(ctx, "UpdateTask", req.TripId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return api.UpdateTask404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	if !h.canEditTasks(role) {
		return api.UpdateTask403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgRoleDenied))}, nil
	}
	patch := model.TaskPatch{Done: req.Body.Done, By: uid}
	if req.Body.Title != nil {
		title := strings.TrimSpace(*req.Body.Title)
		patch.Title = &title
	}
	t, err := h.tasks.Update(ctx, req.TripId, req.TaskId, patch)
	if errors.Is(err, repo.ErrNotFound) {
		return api.UpdateTask404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such to-do."))}, nil
	}
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.UpdateTask400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, msg))}, nil
		}
		return nil, err
	}
	return api.UpdateTask200JSONResponse(taskFrom(t)), nil
}

// DeleteTask removes a to-do. Owners and editors only.
func (h *handlers) DeleteTask(ctx context.Context, req api.DeleteTaskRequestObject) (api.DeleteTaskResponseObject, error) {
	if !h.tasksReady() {
		return api.DeleteTask503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTasks))}, nil
	}
	role, _, ok, err := h.taskAccess(ctx, "DeleteTask", req.TripId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return api.DeleteTask404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	if !h.canEditTasks(role) {
		return api.DeleteTask403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgRoleDenied))}, nil
	}
	if err := h.tasks.Delete(ctx, req.TripId, req.TaskId); err != nil {
		return nil, err
	}
	return api.DeleteTask204Response{}, nil
}

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

const commentsCollection = "comments"

// Comments implements repo.Comments.
type Comments struct {
	client *firestore.Client
}

var _ repo.Comments = (*Comments)(nil)

// NewComments returns the Firestore-backed comment store.
func NewComments(client *firestore.Client) *Comments { return &Comments{client: client} }

type commentDoc struct {
	TargetType string    `firestore:"targetType"`
	TargetID   string    `firestore:"targetId"`
	AuthorUID  string    `firestore:"authorUid"`
	Body       string    `firestore:"body"`
	Mentions   []string  `firestore:"mentions"`
	CreatedAt  time.Time `firestore:"createdAt"`
	EditedAt   time.Time `firestore:"editedAt,omitempty"`
}

func (s *Comments) ref(tripID, id string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(commentsCollection).Doc(id)
}

func commentFromDoc(id string, d commentDoc) model.Comment {
	return model.Comment{ID: id, TargetType: d.TargetType, TargetID: d.TargetID, AuthorUID: d.AuthorUID, Body: d.Body, Mentions: d.Mentions, CreatedAt: d.CreatedAt, EditedAt: d.EditedAt}
}

// Create stores a comment.
func (s *Comments) Create(ctx context.Context, tripID string, c model.Comment) (model.Comment, error) {
	if err := c.Validate(); err != nil {
		return model.Comment{}, err
	}
	if c.Mentions == nil {
		c.Mentions = []string{}
	}
	if _, err := s.ref(tripID, c.ID).Create(ctx, commentDoc{
		TargetType: c.TargetType, TargetID: c.TargetID, AuthorUID: c.AuthorUID, Body: c.Body, Mentions: c.Mentions, CreatedAt: micro(time.Now()),
	}); err != nil {
		return model.Comment{}, mapErr("create comment", err)
	}
	return s.Get(ctx, tripID, c.ID)
}

// Get returns one comment.
func (s *Comments) Get(ctx context.Context, tripID, id string) (model.Comment, error) {
	if tripID == "" || id == "" {
		return model.Comment{}, repo.ErrNotFound
	}
	snap, err := s.ref(tripID, id).Get(ctx)
	if err != nil {
		return model.Comment{}, mapErr("get comment", err)
	}
	var d commentDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Comment{}, mapErr("decode comment", err)
	}
	return commentFromDoc(id, d), nil
}

// List returns the trip's comments, oldest first.
func (s *Comments) List(ctx context.Context, tripID string) ([]model.Comment, error) {
	iter := s.client.Collection(tripsCollection).Doc(tripID).Collection(commentsCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.Comment
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list comments", err)
		}
		var d commentDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode comment", err)
		}
		out = append(out, commentFromDoc(snap.Ref.ID, d))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Edit replaces the body and mentions.
func (s *Comments) Edit(ctx context.Context, tripID, id, body string, mentions []string, now time.Time) (model.Comment, error) {
	if err := model.ValidateCommentText(body, mentions); err != nil {
		return model.Comment{}, err
	}
	if mentions == nil {
		mentions = []string{}
	}
	if _, err := s.ref(tripID, id).Update(ctx, []firestore.Update{
		{Path: "body", Value: body}, {Path: "mentions", Value: mentions}, {Path: "editedAt", Value: micro(now)},
	}); err != nil {
		return model.Comment{}, mapErr("edit comment", err)
	}
	return s.Get(ctx, tripID, id)
}

// Delete removes a comment.
func (s *Comments) Delete(ctx context.Context, tripID, id string) error {
	_, err := s.ref(tripID, id).Delete(ctx)
	return mapErr("delete comment", err)
}

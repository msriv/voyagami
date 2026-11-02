package analytics

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	pubsub "google.golang.org/api/pubsub/v1"
)

// PubSub publishes each row as one JSON message. A BigQuery subscription on the
// topic writes the rows into analytics.events (VOY-229).
type PubSub struct {
	topic string // projects/<project>/topics/<name>
	svc   *pubsub.Service
}

// NewPubSub uses the runtime's own identity (no key).
func NewPubSub(ctx context.Context, project, topic string) (*PubSub, error) {
	svc, err := pubsub.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("pubsub: %w", err)
	}
	return &PubSub{topic: "projects/" + project + "/topics/" + topic, svc: svc}, nil
}

// Add publishes the rows in one request.
func (p *PubSub) Add(ctx context.Context, rows []Record) error {
	msgs, err := Messages(rows)
	if err != nil || len(msgs) == 0 {
		return err
	}
	_, err = p.svc.Projects.Topics.Publish(p.topic, &pubsub.PublishRequest{Messages: msgs}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("publish analytics: %w", err)
	}
	return nil
}

// Messages encodes rows as Pub/Sub messages.
func Messages(rows []Record) ([]*pubsub.PubsubMessage, error) {
	out := make([]*pubsub.PubsubMessage, 0, len(rows))
	for _, r := range rows {
		b, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		out = append(out, &pubsub.PubsubMessage{Data: base64.StdEncoding.EncodeToString(b)})
	}
	return out, nil
}

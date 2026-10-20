package firestorerepo

import (
	"context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

// testClient connects to the Firestore emulator. These tests need it:
// `make test` starts one. Plain `go test` skips them unless REQUIRE_EMULATOR=1,
// which CI sets so a missing emulator fails instead of passing silently.
func testClient(t *testing.T) *firestore.Client {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		if os.Getenv("REQUIRE_EMULATOR") == "1" {
			t.Fatal("FIRESTORE_EMULATOR_HOST is not set but REQUIRE_EMULATOR=1")
		}
		t.Skip("FIRESTORE_EMULATOR_HOST not set; run `make test` to start the Firestore emulator")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := firestore.NewClient(ctx, "demo-voyagami")
	if err != nil {
		t.Fatalf("firestore client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

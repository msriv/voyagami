package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/otp"
	"github.com/msriv/voyagami/services/api/internal/repo/firestorerepo"
	"github.com/msriv/voyagami/services/api/internal/server"
	"github.com/msriv/voyagami/services/api/internal/signin"
)

// signInDeps are the pieces that turn phone sign-in on.
type signInDeps struct {
	verifier auth.Verifier
	options  []server.Option
	close    func()
}

// buildSignIn connects storage and wires the sign-in service. The SMS provider
// is the dev stub in dev; staging and prod have none until P1, so there the
// sign-in routes answer 503 while tokens already issued still verify.
func buildSignIn(ctx context.Context, cfg config.Config, logger *slog.Logger) (signInDeps, error) {
	secret := cfg.AuthSecret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return signInDeps{}, err
		}
		secret = hex.EncodeToString(b)
		logger.Warn("AUTH_SECRET is not set, so sign-in tokens use a random per-process secret and stop working on restart")
	}
	keys, err := signin.NewKeys([]byte(secret))
	if err != nil {
		return signInDeps{}, err
	}

	project := cfg.GCPProject
	if project == "" {
		project = firestore.DetectProjectID
	}
	client, err := firestore.NewClient(ctx, project)
	if err != nil {
		return signInDeps{}, fmt.Errorf("firestore (set GCP_PROJECT): %w", err)
	}

	trips := firestorerepo.NewTrips(client)
	deps := signInDeps{
		verifier: signin.NewVerifier(keys),
		options: []server.Option{
			server.WithProfile(firestorerepo.NewUsers(client)),
			server.WithTrips(trips),
			server.WithEntitlements(entitlements.New(entitlements.ForEnv(cfg.Env)), planSource{firestorerepo.NewEntitlements(client), trips}),
		},
		close: func() { _ = client.Close() },
	}
	if cfg.Env != "dev" {
		logger.Warn("no SMS provider for this environment yet, so phone sign-in is off")
		return deps, nil
	}
	stub, err := otp.NewStub(cfg.Env)
	if err != nil {
		_ = client.Close()
		return signInDeps{}, err
	}
	sms := otp.NewLimited(stub, otp.DefaultLimits)
	svc := signin.NewService(sms, firestorerepo.NewAccounts(client), firestorerepo.NewSessions(client), keys)
	deps.options = append(deps.options, server.WithSignIn(svc))
	return deps, nil
}

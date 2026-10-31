package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/ask"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/export"
	"github.com/msriv/voyagami/services/api/internal/importer"
	"github.com/msriv/voyagami/services/api/internal/otp"
	"github.com/msriv/voyagami/services/api/internal/places"
	"github.com/msriv/voyagami/services/api/internal/rates"
	"github.com/msriv/voyagami/services/api/internal/repo/firestorerepo"
	"github.com/msriv/voyagami/services/api/internal/server"
	"github.com/msriv/voyagami/services/api/internal/signin"
	"github.com/msriv/voyagami/services/api/internal/stays"
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
	users := firestorerepo.NewUsers(client)
	devices := firestorerepo.NewDevices(client)
	bill := firestorerepo.NewBilling(client)
	consents := firestorerepo.NewConsents(client)
	deletions := firestorerepo.NewDeletions(client)
	deps := signInDeps{
		verifier: signin.NewVerifier(keys),
		options: []server.Option{
			server.WithProfile(users),
			server.WithTrips(trips),
			server.WithStops(firestorerepo.NewStops(client)),
			server.WithTasks(firestorerepo.NewTasks(client)),
			server.WithExpenses(firestorerepo.NewExpenses(client)),
			server.WithRates(rates.NewCached(rates.NewFixed())),
			server.WithComments(firestorerepo.NewComments(client)),
			server.WithFeed(firestorerepo.NewFeed(client)),
			server.WithMeetPoints(firestorerepo.NewMeetPoints(client), firestorerepo.NewPresence(client)),
			server.WithPolls(firestorerepo.NewPolls(client)),
			server.WithInvites(firestorerepo.NewInvites(client)),
			server.WithDevices(devices),
			server.WithSponsored(firestorerepo.NewSponsored(client)),
			server.WithConsents(consents),
			server.WithDeletions(deletions),
			server.WithExport(export.Sources{
				Users: users, Consents: consents, Trips: trips, Stops: firestorerepo.NewStops(client),
				Expenses: firestorerepo.NewExpenses(client), Bookings: firestorerepo.NewBookings(client), Journal: firestorerepo.NewJournal(client),
			}),
			server.WithConsents(firestorerepo.NewConsents(client)),
			server.WithSupport(firestorerepo.NewSupport(client)),
			server.WithCatalog(firestorerepo.NewCatalog(client)),
			server.WithStamps(firestorerepo.NewStamps(client)),
			server.WithStampReward(firestorerepo.NewEntitlements(client)),
			server.WithCrewEvents(newCrewEvents(cfg, logger, trips, users, devices)),
			server.WithUsage(firestorerepo.NewUsage(client)),
			server.WithJournal(firestorerepo.NewJournal(client)),
			server.WithBookings(firestorerepo.NewBookings(client)),
			server.WithSettleUp(firestorerepo.NewSettlements(client)),
			server.WithEntitlements(entitlements.New(limitsFor(cfg)), planSource{firestorerepo.NewEntitlements(client), trips, bill, cfg.DevPlan, cfg.Env == "dev"}),
		},
		close: func() { _ = client.Close() },
	}
	if cfg.RevenueCatWebhookSecret != "" {
		deps.options = append(deps.options, server.WithBilling(bill, cfg.RevenueCatWebhookSecret))
	} else {
		logger.Warn("REVENUECAT_WEBHOOK_SECRET is not set, so purchases do not reach plans")
	}
	partners, err := stays.Load(cfg.StayPartners)
	if err != nil {
		return signInDeps{}, err
	}
	deps.options = append(deps.options, server.WithStays(partners, firestorerepo.NewStayClicks(client)))
	expPartners, err := stays.Load(cfg.ExperiencePartners)
	if err != nil {
		return signInDeps{}, err
	}
	deps.options = append(deps.options, server.WithExperiences(expPartners, firestorerepo.NewStayClicks(client)))
	if cfg.JobsSecret != "" {
		deps.options = append(deps.options, server.WithJobs(newTrialReminders(cfg, logger, users, devices, firestorerepo.NewEntitlements(client)), cfg.JobsSecret),
			server.WithDeletionJob(accountPurge{deletions: deletions, logger: logger}, cfg.JobsSecret))
	} else {
		logger.Warn("JOBS_SECRET is not set, so trial reminders and account deletions do not run")
	}
	if cfg.Env == "dev" {
		deps.options = append(deps.options, server.WithDevLogs(firestorerepo.NewDevLogs(client)))
	}
	if cfg.VertexLocation != "" && cfg.GCPProject != "" {
		deps.options = append(deps.options, server.WithImporter(&importer.Vertex{
			Project: cfg.GCPProject, Location: cfg.VertexLocation, Model: cfg.VertexModel,
		}))
	} else {
		logger.Warn("VERTEX_LOCATION or GCP_PROJECT is not set, so reading bookings from text and PDFs is off")
	}
	if cfg.VertexLocation != "" && cfg.GCPProject != "" {
		deps.options = append(deps.options, server.WithAsk(&ask.Vertex{
			Project: cfg.GCPProject, Location: cfg.VertexLocation, Model: cfg.VertexModel,
		}, firestorerepo.NewEntitlements(client)))
	}
	switch {
	case cfg.PlacesAPIKey != "":
		deps.options = append(deps.options, server.WithPlaces(places.Google{Key: cfg.PlacesAPIKey}))
	case cfg.Env == "dev":
		deps.options = append(deps.options, server.WithPlaces(places.OSM{}))
	default:
		logger.Warn("PLACES_API_KEY is not set, so place search is off")
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
	deps.options = append(deps.options, server.WithSignIn(devPlanSignIn{svc, firestorerepo.NewEntitlements(client)}))
	return deps, nil
}

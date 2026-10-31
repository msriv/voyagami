package main

import (
	"testing"

	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
)

func TestMockNumbersAreDistinctAndValid(t *testing.T) {
	seen := map[string]bool{}
	for phone, plan := range devMockAccounts {
		if seen[plan] {
			t.Errorf("plan %q is used twice", plan)
		}
		seen[plan] = true
		if len(phone) != 12 || phone[:8] != "+1555555" {
			t.Errorf("%s is not in the fictional 555 range", phone)
		}
	}
	if len(seen) != 3 {
		t.Errorf("want one mock per plan, got %d", len(seen))
	}
}

func TestLimitsForIsFreeEverywhere(t *testing.T) {
	if limitsFor(config.Config{Env: "dev"})[entitlements.UpcomingTrips] != 2 {
		t.Error("the table must be the Free values; dev opens it per account")
	}
}

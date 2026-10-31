package main

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/signin"
)

const (
	devPlanFree = "free"
	devPlanPlus = "plus"
	devPlanPass = "pass"
)

// devMockAccounts are the dev sign-in numbers that stand for one plan each.
// They use the reserved fictional 555-01xx range, so they never reach a real
// phone. Sign in with any of them and the stub code 123456; every other number
// is an ordinary dev account with no limits.
var devMockAccounts = map[string]string{
	"+15555550101": devPlanFree,
	"+15555550102": devPlanPlus,
	"+15555550103": devPlanPass,
}

// devPlanStore marks an account as a plan mock.
type devPlanStore interface {
	SetDevPlan(ctx context.Context, uid, plan string) error
}

// devPlanSignIn wraps sign-in so a mock number is marked with its plan every
// time it signs in. The mark lives on the account, so the plan follows the
// account, not the server. The pass mock is Free itself, but every trip it
// belongs to counts as having a Trip Pass, so the crew sees that plan too.
type devPlanSignIn struct {
	*signin.Service
	plans devPlanStore
}

func (d devPlanSignIn) Verify(ctx context.Context, phone, code string, dev signin.Device) (signin.Tokens, error) {
	t, err := d.Service.Verify(ctx, phone, code, dev)
	if err != nil {
		return t, err
	}
	if plan, ok := devMockAccounts[phone]; ok {
		if err := d.plans.SetDevPlan(ctx, t.UserID, plan); err != nil {
			return signin.Tokens{}, err
		}
	}
	return t, nil
}

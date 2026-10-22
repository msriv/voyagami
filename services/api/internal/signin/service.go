package signin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/otp"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const (
	// AccessTTL is how long an access token works.
	AccessTTL = 15 * time.Minute
	// RefreshTTL is how long a refresh token works. Each refresh starts it again.
	RefreshTTL = 60 * 24 * time.Hour
)

// ErrInvalidCode means the one-time code is wrong.
var ErrInvalidCode = errors.New("invalid code")

// Tokens is what a successful sign-in or refresh returns.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	UserID       string
	IsNewUser    bool
}

// Device is what the phone tells us about itself at sign-in. It only fills in
// a brand-new account until onboarding asks the person.
type Device struct {
	Region   string // ISO 3166-1 alpha-2, optional
	Language string // BCP 47, optional
}

// Service runs sign-in.
type Service struct {
	SMS      otp.SMSProvider
	Accounts repo.Accounts
	Sessions repo.Sessions
	Keys     Keys

	now func() time.Time
}

// NewService returns a sign-in service.
func NewService(sms otp.SMSProvider, accounts repo.Accounts, sessions repo.Sessions, keys Keys) *Service {
	return &Service{SMS: sms, Accounts: accounts, Sessions: sessions, Keys: keys, now: time.Now}
}

// RequestCode sends a one-time code. It returns otp.ErrInvalidNumber or
// otp.ErrRateLimited, which callers map to 400 and 429.
func (s *Service) RequestCode(ctx context.Context, phone string) error {
	return s.SMS.Send(ctx, phone)
}

// Verify checks the code and signs the person in, creating the account on the
// first success.
func (s *Service) Verify(ctx context.Context, phone, code string, dev Device) (Tokens, error) {
	ok, err := s.SMS.Check(ctx, phone, code)
	if err != nil {
		return Tokens{}, err
	}
	if !ok {
		return Tokens{}, ErrInvalidCode
	}
	uid, err := newID()
	if err != nil {
		return Tokens{}, err
	}
	u, created, err := s.Accounts.FindOrCreate(ctx, s.Keys.PhoneHash(phone), newUser(uid, dev))
	if err != nil {
		return Tokens{}, fmt.Errorf("find account: %w", err)
	}
	t, err := s.startSession(ctx, u.ID)
	t.IsNewUser = created
	return t, err
}

// Refresh swaps a refresh token for a new pair. A token that was already
// swapped ends the session and returns auth.ErrInvalidToken.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	sid, secret, ok := strings.Cut(refreshToken, ".")
	if !ok || sid == "" || secret == "" {
		return Tokens{}, auth.ErrInvalidToken
	}
	sess, err := s.Sessions.Get(ctx, sid)
	if errors.Is(err, repo.ErrNotFound) {
		return Tokens{}, auth.ErrInvalidToken
	}
	if err != nil {
		return Tokens{}, err
	}
	newSecret, err := randomSecret()
	if err != nil {
		return Tokens{}, err
	}
	err = s.Sessions.Rotate(ctx, sid, hashSecret(secret), hashSecret(newSecret), s.now().Add(RefreshTTL))
	if errors.Is(err, repo.ErrNotFound) || errors.Is(err, repo.ErrTokenReuse) {
		return Tokens{}, auth.ErrInvalidToken
	}
	if err != nil {
		return Tokens{}, err
	}
	return s.tokens(sess.UserID, sid, newSecret), nil
}

// SignOut ends one session.
func (s *Service) SignOut(ctx context.Context, sessionID string) error {
	return s.Sessions.Revoke(ctx, sessionID)
}

func (s *Service) startSession(ctx context.Context, userID string) (Tokens, error) {
	sid, err := newID()
	if err != nil {
		return Tokens{}, err
	}
	secret, err := randomSecret()
	if err != nil {
		return Tokens{}, err
	}
	err = s.Sessions.Create(ctx, model.Session{
		ID: sid, UserID: userID, RefreshHash: hashSecret(secret), ExpiresAt: s.now().Add(RefreshTTL),
	})
	if err != nil {
		return Tokens{}, fmt.Errorf("create session: %w", err)
	}
	return s.tokens(userID, sid, secret), nil
}

func (s *Service) tokens(userID, sessionID, secret string) Tokens {
	return Tokens{
		AccessToken:  s.Keys.issueAccess(userID, sessionID, s.now().Add(AccessTTL)),
		RefreshToken: sessionID + "." + secret,
		ExpiresIn:    AccessTTL,
		UserID:       userID,
	}
}

// newUser is the account made on first sign-in. Region and currency use the
// ISO "unknown" codes (ZZ, XXX) because the person has not told us yet: nothing
// here assumes a country. Onboarding replaces them. Units are the ISO 8601 and
// metric defaults until then.
func newUser(id string, dev Device) model.User {
	region := "ZZ"
	if len(dev.Region) == 2 {
		region = strings.ToUpper(dev.Region)
	}
	lang := dev.Language
	if lang == "" {
		lang = "en"
	}
	return model.User{
		ID:           id,
		HomeRegion:   region,
		HomeCurrency: "XXX",
		Language:     lang,
		RegionSource: "device",
		Units:        model.Units{Distance: "km", Temperature: "c", Clock: "24h", FirstWeekday: 1},
	}
}

func newID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

package signin

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/otp"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const phone = "+919876543210"

type memAccounts struct {
	mu    sync.Mutex
	users map[string]model.User // by phone hash
}

func (m *memAccounts) FindOrCreate(_ context.Context, hash string, nu model.User) (model.User, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[hash]; ok {
		return u, false, nil
	}
	m.users[hash] = nu
	return nu, true, nil
}

type memSessions struct {
	mu sync.Mutex
	m  map[string]*model.Session
}

func (s *memSessions) Create(_ context.Context, x model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[x.ID]; ok {
		return repo.ErrAlreadyExists
	}
	s.m[x.ID] = &x
	return nil
}

func (s *memSessions) Get(_ context.Context, id string) (model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if x, ok := s.m[id]; ok {
		return *x, nil
	}
	return model.Session{}, repo.ErrNotFound
}

func (s *memSessions) Rotate(_ context.Context, id, oldHash, newHash string, exp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.m[id]
	if !ok || x.RevokedAt != nil {
		return repo.ErrNotFound
	}
	if x.RefreshHash != oldHash {
		now := time.Now()
		x.RevokedAt = &now
		return repo.ErrTokenReuse
	}
	x.RefreshHash, x.ExpiresAt = newHash, exp
	return nil
}

func (s *memSessions) Revoke(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if x, ok := s.m[id]; ok {
		now := time.Now()
		x.RevokedAt = &now
	}
	return nil
}

func newTestService(t *testing.T) (*Service, *memAccounts, *memSessions) {
	t.Helper()
	keys, err := NewKeys([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	stub, _ := otp.NewStub("dev")
	acc := &memAccounts{users: map[string]model.User{}}
	ses := &memSessions{m: map[string]*model.Session{}}
	return NewService(stub, acc, ses, keys), acc, ses
}

func TestVerifyCreatesAccountOnce(t *testing.T) {
	svc, acc, _ := newTestService(t)
	ctx := context.Background()

	first, err := svc.Verify(ctx, phone, otp.DevCode, Device{Region: "in", Language: "hi"})
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if !first.IsNewUser || first.UserID == "" || first.AccessToken == "" || first.RefreshToken == "" {
		t.Fatalf("first verify = %+v", first)
	}
	second, err := svc.Verify(ctx, phone, otp.DevCode, Device{})
	if err != nil {
		t.Fatalf("second verify: %v", err)
	}
	if second.IsNewUser || second.UserID != first.UserID {
		t.Errorf("second verify = %+v, want same user and not new", second)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("each sign-in must get its own session")
	}

	// The number is stored only as a keyed hash.
	for hash, u := range acc.users {
		if strings.Contains(hash, phone) || hash == phone {
			t.Error("phone number stored in clear")
		}
		if u.HomeRegion != "IN" || u.Language != "hi" {
			t.Errorf("new user locale = %s/%s, want IN/hi from the device", u.HomeRegion, u.Language)
		}
		if err := u.Validate(); err != nil {
			t.Errorf("new user is invalid: %v", err)
		}
	}
}

func TestNewUserWithoutDeviceInfoAssumesNoCountry(t *testing.T) {
	svc, acc, _ := newTestService(t)
	if _, err := svc.Verify(context.Background(), phone, otp.DevCode, Device{}); err != nil {
		t.Fatal(err)
	}
	for _, u := range acc.users {
		if u.HomeRegion != "ZZ" || u.HomeCurrency != "XXX" || u.Language != "en" {
			t.Errorf("defaults = %s/%s/%s, want ZZ/XXX/en", u.HomeRegion, u.HomeCurrency, u.Language)
		}
	}
}

func TestVerifyErrors(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Verify(ctx, phone, "000000", Device{}); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("wrong code = %v, want ErrInvalidCode", err)
	}
	if _, err := svc.Verify(ctx, "nope", otp.DevCode, Device{}); !errors.Is(err, otp.ErrInvalidNumber) {
		t.Errorf("bad number = %v, want ErrInvalidNumber", err)
	}
}

func TestAccessTokenVerifies(t *testing.T) {
	svc, _, _ := newTestService(t)
	tok, err := svc.Verify(context.Background(), phone, otp.DevCode, Device{})
	if err != nil {
		t.Fatal(err)
	}
	v := NewVerifier(svc.Keys)
	id, err := v.Verify(context.Background(), tok.AccessToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if id.UserID != tok.UserID || id.SessionID == "" {
		t.Errorf("identity = %+v", id)
	}

	// Expired.
	v.now = func() time.Time { return time.Now().Add(AccessTTL + time.Second) }
	if _, err := v.Verify(context.Background(), tok.AccessToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("expired = %v, want ErrInvalidToken", err)
	}
}

func TestAccessTokenRejectsTampering(t *testing.T) {
	svc, _, _ := newTestService(t)
	tok, _ := svc.Verify(context.Background(), phone, otp.DevCode, Device{})
	other, _ := NewKeys([]byte(strings.Repeat("z", 32)))
	parts := strings.Split(tok.AccessToken, ".")
	cases := map[string]struct {
		v     Verifier
		token string
	}{
		"other key":        {NewVerifier(other), tok.AccessToken},
		"empty":            {NewVerifier(svc.Keys), ""},
		"two parts":        {NewVerifier(svc.Keys), parts[0] + "." + parts[1]},
		"changed claims":   {NewVerifier(svc.Keys), parts[0] + "." + b64.EncodeToString([]byte(`{"sub":"x","sid":"y","exp":9999999999}`)) + "." + parts[2]},
		"unknown version":  {NewVerifier(svc.Keys), "v2." + parts[1] + "." + parts[2]},
		"not base64":       {NewVerifier(svc.Keys), "v1.!!.!!"},
		"refresh as token": {NewVerifier(svc.Keys), tok.RefreshToken},
	}
	for name, c := range cases {
		if _, err := c.v.Verify(context.Background(), c.token); !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestRefreshRotatesAndDetectsReuse(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	first, _ := svc.Verify(ctx, phone, otp.DevCode, Device{})

	second, err := svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if second.UserID != first.UserID || second.IsNewUser || second.RefreshToken == first.RefreshToken {
		t.Errorf("refresh = %+v", second)
	}

	// Replaying the old token fails and ends the session, even for the new token.
	if _, err := svc.Refresh(ctx, first.RefreshToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("replay = %v, want ErrInvalidToken", err)
	}
	if _, err := svc.Refresh(ctx, second.RefreshToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("after reuse, new token = %v, want ErrInvalidToken", err)
	}
}

func TestRefreshRejectsGarbage(t *testing.T) {
	svc, _, _ := newTestService(t)
	for _, tok := range []string{"", "abc", ".", "a.", ".b", "missing-session.secret"} {
		if _, err := svc.Refresh(context.Background(), tok); !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("Refresh(%q) = %v, want ErrInvalidToken", tok, err)
		}
	}
}

func TestSignOutEndsSession(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	tok, _ := svc.Verify(ctx, phone, otp.DevCode, Device{})
	id, _ := NewVerifier(svc.Keys).Verify(ctx, tok.AccessToken)

	if err := svc.SignOut(ctx, id.SessionID); err != nil {
		t.Fatal(err)
	}
	if err := svc.SignOut(ctx, id.SessionID); err != nil {
		t.Errorf("second sign-out = %v, want nil", err)
	}
	if _, err := svc.Refresh(ctx, tok.RefreshToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("refresh after sign-out = %v, want ErrInvalidToken", err)
	}
}

func TestPhoneHashIsKeyed(t *testing.T) {
	a, _ := NewKeys([]byte(strings.Repeat("a", 32)))
	b, _ := NewKeys([]byte(strings.Repeat("b", 32)))
	if a.PhoneHash(phone) == b.PhoneHash(phone) {
		t.Error("different secrets must give different hashes")
	}
	again := a.PhoneHash(phone)
	if a.PhoneHash(phone) != again || again == a.PhoneHash(phone+"1") {
		t.Error("hash must be stable and distinguish numbers")
	}
	if strings.Contains(a.PhoneHash(phone), "9876") {
		t.Error("hash leaks digits")
	}
}

func TestNewKeysRejectsShortSecret(t *testing.T) {
	if _, err := NewKeys([]byte("short")); err == nil {
		t.Error("want error for a short secret")
	}
}

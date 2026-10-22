package signin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/msriv/voyagami/services/api/internal/auth"
)

// accessPrefix names the token format so it can change later.
const accessPrefix = "v1"

type accessClaims struct {
	Sub string `json:"sub"` // user ID
	Sid string `json:"sid"` // session ID
	Exp int64  `json:"exp"` // unix seconds
}

var b64 = base64.RawURLEncoding

// issueAccess signs a short-lived access token: v1.<claims>.<mac>.
func (k Keys) issueAccess(userID, sessionID string, exp time.Time) string {
	claims, _ := json.Marshal(accessClaims{Sub: userID, Sid: sessionID, Exp: exp.Unix()})
	body := accessPrefix + "." + b64.EncodeToString(claims)
	return body + "." + b64.EncodeToString(k.mac(body))
}

func (k Keys) mac(body string) []byte {
	m := hmac.New(sha256.New, k.access)
	m.Write([]byte(body))
	return m.Sum(nil)
}

// Verifier checks access tokens. It is stateless: a signed-out session's
// access token stays valid until it expires (15 minutes), which is the cost of
// not reading storage on every request.
type Verifier struct {
	keys Keys
	now  func() time.Time
}

var _ auth.Verifier = Verifier{}

// NewVerifier returns the access-token verifier.
func NewVerifier(k Keys) Verifier { return Verifier{keys: k, now: time.Now} }

// Verify returns the Identity in the token, or auth.ErrInvalidToken.
func (v Verifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != accessPrefix {
		return auth.Identity{}, auth.ErrInvalidToken
	}
	got, err := b64.DecodeString(parts[2])
	if err != nil || !hmac.Equal(got, v.keys.mac(parts[0]+"."+parts[1])) {
		return auth.Identity{}, auth.ErrInvalidToken
	}
	raw, err := b64.DecodeString(parts[1])
	if err != nil {
		return auth.Identity{}, auth.ErrInvalidToken
	}
	var c accessClaims
	if json.Unmarshal(raw, &c) != nil || c.Sub == "" || c.Sid == "" || !v.now().Before(time.Unix(c.Exp, 0)) {
		return auth.Identity{}, auth.ErrInvalidToken
	}
	return auth.Identity{UserID: c.Sub, SessionID: c.Sid}, nil
}

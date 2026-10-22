// Package signin turns a verified phone number into a signed-in session:
// one-time codes in, short-lived access tokens and rotating refresh tokens out.
package signin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// minSecretBytes is the shortest master secret accepted.
const minSecretBytes = 32

// Keys are the secrets derived from the one master secret (AUTH_SECRET), so
// access tokens and phone hashes never share a key.
type Keys struct {
	access []byte
	phone  []byte
}

// NewKeys derives the keys. It rejects a secret under 32 bytes.
func NewKeys(master []byte) (Keys, error) {
	if len(master) < minSecretBytes {
		return Keys{}, errors.New("auth secret must be at least 32 bytes")
	}
	return Keys{access: derive(master, "voyagami/access-token/v1"), phone: derive(master, "voyagami/phone-hash/v1")}, nil
}

func derive(master []byte, label string) []byte {
	m := hmac.New(sha256.New, master)
	m.Write([]byte(label))
	return m.Sum(nil)
}

// PhoneHash returns the keyed hash stored in place of a phone number. A plain
// hash would be easy to reverse by trying every number, so the key matters.
func (k Keys) PhoneHash(e164 string) string {
	m := hmac.New(sha256.New, k.phone)
	m.Write([]byte(e164))
	return hex.EncodeToString(m.Sum(nil))
}

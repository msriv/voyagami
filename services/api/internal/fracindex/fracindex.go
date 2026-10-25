// Package fracindex makes sortable keys that fit between two other keys, so
// moving one item rewrites one record. Keys are base-62 fractions compared as
// plain strings, and a key never ends in the lowest digit.
package fracindex

import (
	"errors"
	"strings"
)

const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ErrOrder means a was not before b.
var ErrOrder = errors.New("fracindex: a must sort before b")

// Between returns a key that sorts after a and before b. An empty a means the
// start of the list and an empty b means the end.
func Between(a, b string) (string, error) {
	if !valid(a) || !valid(b) {
		return "", errors.New("fracindex: invalid key")
	}
	if b != "" && a >= b {
		return "", ErrOrder
	}
	return mid(a, b), nil
}

func valid(k string) bool {
	if strings.TrimLeft(k, digits) != "" {
		return false
	}
	return !strings.HasSuffix(k, digits[:1])
}

// mid works on keys where a < b (or b is the end), after trimming the shared prefix.
func mid(a, b string) string {
	if b != "" {
		n := 0
		for n < len(b) && digitAt(a, n) == b[n] {
			n++
		}
		if n > 0 {
			return b[:n] + mid(tail(a, n), b[n:])
		}
	}
	lo, hi := 0, len(digits)
	if a != "" {
		lo = strings.IndexByte(digits, a[0])
	}
	if b != "" {
		hi = strings.IndexByte(digits, b[0])
	}
	if hi-lo > 1 {
		return string(digits[(lo+hi+1)/2])
	}
	if b != "" && len(b) > 1 {
		return b[:1]
	}
	return string(digits[lo]) + mid(tail(a, 1), "")
}

func digitAt(a string, i int) byte {
	if i < len(a) {
		return a[i]
	}
	return digits[0]
}

func tail(a string, n int) string {
	if n >= len(a) {
		return ""
	}
	return a[n:]
}

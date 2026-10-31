// Package stays builds hand-off links to stay partners. The partners come from
// configuration, so no partner name or ID is written in code. The app sends
// people to the partner to book; no booking or payment happens here.
package stays

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// Partner describes one booking site and how to write a search link for it.
type Partner struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	// Params names the partner's query parameters. An empty name leaves that
	// piece of the search out.
	Params Params `json:"params"`
	// Fixed query parameters, such as the affiliate tracking id.
	Fixed map[string]string `json:"fixed"`
}

// Params holds the partner's query parameter names.
type Params struct {
	Destination string `json:"destination"`
	CheckIn     string `json:"checkIn"`
	CheckOut    string `json:"checkOut"`
	Guests      string `json:"guests"`
}

// Search is what the person is looking for. Dates are YYYY-MM-DD and optional.
type Search struct {
	Destination string
	CheckIn     string
	CheckOut    string
	Guests      int
}

// Load reads the partner list from JSON and rejects a list that cannot work.
// An empty string means no partners.
func Load(raw string) ([]Partner, error) {
	if raw == "" {
		return nil, nil
	}
	var ps []Partner
	if err := json.Unmarshal([]byte(raw), &ps); err != nil {
		return nil, fmt.Errorf("stay partners: %w", err)
	}
	seen := map[string]bool{}
	for _, p := range ps {
		u, err := url.Parse(p.BaseURL)
		switch {
		case p.ID == "" || p.Name == "":
			return nil, errors.New("stay partners: every partner needs an id and a name")
		case seen[p.ID]:
			return nil, fmt.Errorf("stay partners: duplicate id %q", p.ID)
		case err != nil || u.Scheme != "https" || u.Host == "":
			return nil, fmt.Errorf("stay partners: %q needs an https baseUrl", p.ID)
		}
		seen[p.ID] = true
	}
	return ps, nil
}

// Build writes the hand-off link: the partner's address, the search and the
// fixed tracking parameters. It adds nothing about the person.
func Build(p Partner, s Search) (string, error) {
	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	set := func(name, value string) {
		if name != "" && value != "" {
			q.Set(name, value)
		}
	}
	set(p.Params.Destination, s.Destination)
	set(p.Params.CheckIn, s.CheckIn)
	set(p.Params.CheckOut, s.CheckOut)
	if s.Guests > 0 {
		set(p.Params.Guests, strconv.Itoa(s.Guests))
	}
	for k, v := range p.Fixed {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

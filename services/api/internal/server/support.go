package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithSupport turns on the Help screen's contact form. Without it the route answers 503.
func WithSupport(s repo.Support) Option {
	return func(h *handlers) { h.support, h.supportLimit = s, &userLimiter{every: 12 * time.Minute, burst: 5} }
}

const msgNoSupport = "Support is not available right now."

// userLimiter caps how often one person may do something.
//
// shortcut: counted per server instance and forgotten on restart, so it stops a
// stuck button, not a determined abuser; move to a shared counter when abuse shows.
type userLimiter struct {
	every time.Duration
	burst int
	mu    sync.Mutex
	by    map[string]*rate.Limiter
}

func (l *userLimiter) allow(uid string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.by == nil || len(l.by) > 10000 {
		l.by = map[string]*rate.Limiter{}
	}
	lim, ok := l.by[uid]
	if !ok {
		lim = rate.NewLimiter(rate.Every(l.every), l.burst)
		l.by[uid] = lim
	}
	return lim.Allow()
}

// SendSupportMessage stores a message from the Help screen.
func (h *handlers) SendSupportMessage(ctx context.Context, req api.SendSupportMessageRequestObject) (api.SendSupportMessageResponseObject, error) {
	id, err := identity(ctx, "SendSupportMessage")
	if err != nil {
		return nil, err
	}
	if h.support == nil {
		return api.SendSupportMessage503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSupport))}, nil
	}
	b := req.Body
	m := model.SupportMessage{UserID: id.UserID, Category: string(b.Category), Text: trimmed(&b.Text), CreatedAt: time.Now()}
	if d := b.Diagnostics; d != nil {
		m.Diagnostics = model.Diagnostics{AppVersion: trimmed(d.AppVersion), OSVersion: trimmed(d.OsVersion), DeviceModel: trimmed(d.DeviceModel), Locale: trimmed(d.Locale), RequestID: trimmed(d.RequestId)}
		if d.Os != nil {
			m.Diagnostics.OS = string(*d.Os)
		}
	}
	if err := m.Validate(); err != nil {
		var invalid *model.ValidationError
		if errors.As(err, &invalid) {
			return api.SendSupportMessage400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, invalid.Error())}, nil
		}
		return nil, err
	}
	if !h.supportLimit.allow(id.UserID) {
		return api.SendSupportMessage429JSONResponse{TooManyRequestsJSONResponse: api.TooManyRequestsJSONResponse(errBody(ctx, codeRateLimited, "Too many messages. Try again later."))}, nil
	}
	mid, err := h.support.Add(ctx, m)
	if err != nil {
		return nil, err
	}
	return api.SendSupportMessage202JSONResponse{Id: mid}, nil
}

package dto

import (
	"errors"
	"time"
)

// Максимальный срок жизни ссылки — 7 дней (в минутах).
const maxShareTTLMinutes = 7 * 24 * 60

type ShareRequest struct {
	TTLMinutes int `json:"ttl_minutes"` // сколько минут будет работать ссылка
}

func (r ShareRequest) Validate() error {
	if r.TTLMinutes < 1 || r.TTLMinutes > maxShareTTLMinutes {
		return errors.New("ttl_minutes must be between 1 and 10080 (7 days)")
	}
	return nil
}

type ShareResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

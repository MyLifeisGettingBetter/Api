package types

import (
	"time"
)

type User struct {
	ID                int       `toml:"id"`
	UID               string    `toml:"uid"`
	Name              string    `toml:"name"`
	Email             string    `toml:"email"`
	APIKey            string    `toml:"api_key"`
	AllowedModels     []string  `toml:"allowed_models"`
	Enable            bool      `toml:"enable"`
	DailyRequestLimit int       `toml:"daily_request_limit"`
	DailyRequestUsed  int       `toml:"daily_request_used"`
	DailyTokenLimit   int       `toml:"daily_token_limit"`
	DailyTokenUsed    int       `toml:"daily_token_used"`
	CreateAt          time.Time `toml:"create_at"`
}

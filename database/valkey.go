package database

import (
	"errors"

	"github.com/EvieePy/Echo/models"
	"github.com/valkey-io/valkey-go"
)

const VK_RATELIMIT_SCRIPT = `
local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window_seconds = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local member = ARGV[4]

local window_start = now - window_seconds * 1000

redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

local count = redis.call('ZCARD', key)

if count < max_requests then
  redis.call('ZADD', key, now, member)
  redis.call('EXPIRE', key, window_seconds)
  return { 1, max_requests - count - 1, 0 }
end

local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local retry_after_ms = window_seconds * 1000

if #oldest >= 2 then
  retry_after_ms = oldest[2] + window_seconds * 1000 - now
end

return { 0, 0, retry_after_ms }
`

type Valkey struct {
	Client valkey.Client
	Lua    *valkey.Lua
}

func NewValkey(config *models.Config) (*Valkey, error) {
	if config.Cache.DSN == "" {
		return nil, errors.New("cache config is missing DSN.")
	}

	addr := config.Cache.DSN
	client, err := valkey.NewClient(valkey.MustParseURL(addr))

	if err != nil {
		panic(err)
	}

	lua := valkey.NewLuaScript(VK_RATELIMIT_SCRIPT)
	return &Valkey{Client: client, Lua: lua}, nil
}

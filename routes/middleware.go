package routes

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/EvieePy/Echo/models"
	"github.com/EvieePy/Echo/state"
	"github.com/labstack/echo/v5"
)

type MiddlewareView struct {
	ctx *state.Context
}

func (v *MiddlewareView) LoadRoutes() {
	v.ctx.Server.Use(v.rateLimiter)
}

func (v *MiddlewareView) rateLimiter(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		clientIP := c.RealIP()

		// Check if we have valkey and if the request is not local...
		// TODO: ...
		// if clientIP == "::1" {
		// 	return next(c)
		// }
		if v.ctx.Valkey == nil {
			return next(c)
		}

		// Find our limits for this path...
		request := c.Request()
		var matched *models.RateLimitT

		for _, limit := range v.ctx.Config.Limits {
			if limit.Method == request.Method && limit.Route == c.Path() {
				matched = &limit
			}
		}
		if matched == nil {
			return next(c)
		}

		now := time.Now().UTC()
		nowS := strconv.FormatInt(now.UnixMilli(), 10)
		nowTs := now.UnixMilli()
		rateS := strconv.Itoa(matched.Rate)
		perS := strconv.Itoa(matched.Per)

		// Random number for timestamp in Valkey...
		max := big.NewInt(1000)
		randInt, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		timestampS := fmt.Sprintf("%s:%d", strconv.FormatInt(nowTs, 10), randInt)

		clientKey := fmt.Sprintf("ratelimit:%s:%s:%s", request.Method, c.Path(), clientIP)
		result := v.ctx.Valkey.Lua.Exec(
			request.Context(),
			v.ctx.Valkey.Client,
			[]string{clientKey},
			[]string{rateS, perS, nowS, timestampS})

		rslice, err := result.AsIntSlice()
		if err != nil {
			v.ctx.Logger.Errorf("rate limiter failed: %v", err)
			return next(c)
		}

		allowed := rslice[0] == 1
		remaining := rslice[1]
		retry := (rslice[2] + 999) / 1000

		c.Response().Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Response().Header().Set("X-RateLimit-Limit", rateS)
		c.Response().Header().Set("X-RateLimit-Retry-After", strconv.FormatInt(retry, 10))

		if !allowed {
			return echo.NewHTTPError(429, "You are requesting too fast. Check headers for ratelimit information.")
		}

		return next(c)
	}
}

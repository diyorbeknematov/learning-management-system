// Package middleware holds the steps every request goes through before its
// handler: request id, logging, panic recovery, CORS, authentication and
// authorization.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDKey = "request_id"
	actorKey     = "actor"
)

// RequestID gives every request an id, takes the one of the client when it is
// a valid UUID, and sends it back in X-Request-ID, so a user can quote it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.NewString()
		}

		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger writes one line per request. A failed request (5xx) is logged as an
// error by response.Fail with the cause, so here it is only the summary.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		c.Next()

		status := c.Writer.Status()

		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		}

		attrs := []any{
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", status,
			"duration", time.Since(started).String(),
			"request_id", c.GetString(requestIDKey),
			"ip", c.ClientIP(),
		}

		if actor, ok := c.Get(actorKey); ok {
			attrs = append(attrs, "user_id", actor.(models.Actor).UserID.String())
		}

		log.Log(c.Request.Context(), level, "request", attrs...)
	}
}

// Recover turns a panic into a 500 answer, so one broken request cannot stop
// the server, and logs the stack.
func Recover(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.ErrorContext(
					c.Request.Context(),
					"panic",
					"panic", recovered,
					"request_id", c.GetString(requestIDKey),
					"stack", string(debug.Stack()),
				)

				response.Error(c, http.StatusInternalServerError, response.CodeInternal, "internal server error")
			}
		}()

		c.Next()
	}
}

// CORS lets the frontends in origins call the API from a browser. An origin
// that is not in the list gets no CORS headers, so the browser blocks it.
func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && (allowed[origin] || allowed["*"]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID, Content-Disposition")
			c.Header("Access-Control-Max-Age", "600")
		}

		// the browser asks first (preflight) before the real request
		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			c.AbortWithStatus(http.StatusNoContent)

			return
		}

		c.Next()
	}
}

// Authenticate reads the access token from "Authorization: Bearer <token>" and
// puts the Actor (who the user is and their role) into the request. A token
// that is sent but wrong or expired is always 401, so the client knows to
// refresh it. When no token is sent, a route with required=false goes on with
// an empty Actor (a visitor), for pages that are open to everybody but show
// more to their owner.
//
// revocations (it may be nil) says whether the tokens of the user were taken
// back since the token was issued, for example because the user was blocked.
// When it cannot answer, the request is refused: letting it through would
// open the door to a user who was just thrown out.
func Authenticate(tokens *token.Manager, revocations Revocations, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		if header == "" {
			if required {
				response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "authentication is required")

				return
			}

			c.Set(actorKey, models.Actor{})
			c.Next()

			return
		}

		scheme, raw, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(raw) == "" {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid authorization header")

			return
		}

		claims, err := tokens.ParseAccessToken(strings.TrimSpace(raw))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")

			return
		}

		if revocations != nil {
			// a token without a time is treated as the oldest one
			var issuedAt time.Time
			if claims.IssuedAt != nil {
				issuedAt = claims.IssuedAt.Time
			}

			revoked, err := revocations.IsRevoked(c.Request.Context(), claims.UserID, issuedAt)
			if err != nil {
				response.Fail(c, err)

				return
			}

			if revoked {
				response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "the session has ended, log in again")

				return
			}
		}

		c.Set(actorKey, models.Actor{UserID: claims.UserID, RoleName: claims.RoleName})
		c.Next()
	}
}

// Revocations tells whether the access tokens of a user, issued at issuedAt,
// were taken back.
type Revocations interface {
	IsRevoked(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error)
}

// Limiter counts hits in a window (Redis does it).
type Limiter interface {
	Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
}

// RateLimit allows each client address max requests in window on this route
// (name tells the routes apart) and answers 429 with Retry-After for the rest.
// A nil limiter turns the limit off. When the limiter cannot be reached the
// request goes on (and the problem is logged): a broken counter must not stop
// the whole API.
func RateLimit(limiter Limiter, log *slog.Logger, name string, max int, window time.Duration) gin.HandlerFunc {
	if limiter == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		count, left, err := limiter.Incr(c.Request.Context(), "rate:"+name+":"+c.ClientIP(), window)
		if err != nil {
			log.WarnContext(c.Request.Context(), "the rate limit could not be checked", "route", name, "error", err)
			c.Next()

			return
		}

		if count > int64(max) {
			seconds := int((left + time.Second - 1) / time.Second)

			c.Header("Retry-After", strconv.Itoa(seconds))
			response.Error(
				c,
				http.StatusTooManyRequests,
				response.CodeTooManyRequests,
				fmt.Sprintf("too many requests, try again in %d seconds", seconds),
			)

			return
		}

		c.Next()
	}
}

// Authorize asks Casbin whether the role of the user may call this route with
// this method. The route is the pattern ("/api/v1/courses/:courseId"), not the
// real address, so the policy lists each route once.
func Authorize(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := ActorFrom(c)

		allowed, err := enforcer.Enforce(actor.RoleName, c.FullPath(), c.Request.Method)
		if err != nil {
			response.Fail(c, err)

			return
		}

		if !allowed {
			response.Error(c, http.StatusForbidden, response.CodeForbidden, "you do not have permission to do this")

			return
		}

		c.Next()
	}
}

// ActorFrom returns the user of the request. For a route without a token it is
// the empty Actor.
func ActorFrom(c *gin.Context) models.Actor {
	actor, _ := c.Get(actorKey)

	value, _ := actor.(models.Actor)

	return value
}

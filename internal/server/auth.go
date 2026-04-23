package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

// ==========================
// JWT Implementation
// ==========================

type JWT struct {
	SigningKey []byte
}

type CustomClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func NewJWT() *JWT {
	return &JWT{
		SigningKey: []byte("GATEWAY"), // TODO: Move to config
	}
}

func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

func (j *JWT) ParserToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token invalid")
}

// ==========================
// Middleware
// ==========================

func JWTAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("token")
		if token == "" {
			// Also check Authorization header Bearer token
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Check Query Param (for WebSockets)
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "1",
				"message": "请求未携带token，无权限访问",
				"data":    "",
			})
		}

		j := NewJWT()
		claims, err := j.ParserToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "1",
				"message": "登录已经过期，请重新登录", // Simplify error message for user
				"data":    "",
			})
		}

		c.Locals("claims", claims)
		return c.Next()
	}
}

// ==========================
// Nonce & Rate Limiting
// ==========================

var (
	nonceStore sync.Map
	nonceMax   = 100000
)

var nonceLimiters sync.Map

func GetLimiter(ip string) *rate.Limiter {
	if l, ok := nonceLimiters.Load(ip); ok {
		return l.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(2, 5) // 2 requests per second, burst 5
	nonceLimiters.Store(ip, limiter)
	return limiter
}

func GenerateNonce() (string, error) {
	// Simple size check to prevent memory exhaustion
	size := 0
	nonceStore.Range(func(_, _ any) bool {
		size++
		return true
	})
	if size > nonceMax {
		return "", fmt.Errorf("nonce store full")
	}

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	nonce := hex.EncodeToString(b)
	nonceStore.Store(nonce, time.Now().Add(2*time.Minute))
	return nonce, nil
}

func ValidateAndConsumeNonce(nonce string) bool {
	v, ok := nonceStore.Load(nonce)
	if !ok {
		return false
	}

	expire := v.(time.Time)
	if time.Now().After(expire) {
		nonceStore.Delete(nonce)
		return false
	}

	nonceStore.Delete(nonce)
	return true
}

func init() {
	// Background cleanup for nonces
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			nonceStore.Range(func(key, value any) bool {
				expire := value.(time.Time)
				if time.Now().After(expire) {
					nonceStore.Delete(key)
				}
				return true
			})
		}
	}()
}

// ==========================
// Brute-force Protection
// ==========================

const (
	MaxLoginFailCount  = 10
	LoginBlockDuration = 3 * time.Minute
)

type LoginFailInfo struct {
	Count      int
	LastFailAt time.Time
	BlockUntil time.Time
}

var (
	loginFailMap   = make(map[string]*LoginFailInfo)
	loginFailMutex sync.Mutex
)

func IsIPBlocked(ip string) (bool, time.Duration) {
	loginFailMutex.Lock()
	defer loginFailMutex.Unlock()

	info, exists := loginFailMap[ip]
	if !exists {
		return false, 0
	}

	if time.Now().Before(info.BlockUntil) {
		return true, time.Until(info.BlockUntil)
	}

	return false, 0
}

func AddLoginFail(ip string) {
	loginFailMutex.Lock()
	defer loginFailMutex.Unlock()

	info, exists := loginFailMap[ip]
	if !exists {
		loginFailMap[ip] = &LoginFailInfo{
			Count:      1,
			LastFailAt: time.Now(),
		}
		return
	}

	info.Count++
	info.LastFailAt = time.Now()

	if info.Count >= MaxLoginFailCount {
		info.BlockUntil = time.Now().Add(LoginBlockDuration)
	}
}

func ClearLoginFail(ip string) {
	loginFailMutex.Lock()
	defer loginFailMutex.Unlock()
	delete(loginFailMap, ip)
}

package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var jwtSecret = []byte("secret-key")

// In-memory users for simplicity
var users = map[string]struct {
	Password string
	Role     string // "viewer" or "admin"
}{
	"viewer": {Password: "viewer123", Role: "viewer"},
	"admin":  {Password: "admin123", Role: "admin"},
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type loginResp struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

func LoginHandler(c echo.Context) error {
	var req loginReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}
	u, ok := users[req.Username]
	if !ok || u.Password != req.Password {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}
	claims := jwt.MapClaims{
		"sub":  req.Username,
		"role": u.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := t.SignedString(jwtSecret)
	return c.JSON(http.StatusOK, loginResp{Token: s, Role: u.Role})
}

func RequireRole(roles ...string) echo.MiddlewareFunc {
	roleSet := map[string]struct{}{}
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authz := c.Request().Header.Get("Authorization")
			if len(authz) < 8 || authz[:7] != "Bearer " {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			}
			tokenStr := authz[7:]
			tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) { return jwtSecret, nil })
			if err != nil || !tok.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}
			claims, ok := tok.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}
			if len(roleSet) == 0 {
				return next(c)
			}
			role, _ := claims["role"].(string)
			if _, ok := roleSet[role]; !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
			}
			return next(c)
		}
	}
}

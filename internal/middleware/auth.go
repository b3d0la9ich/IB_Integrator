package middleware

import (
	"net/http"

	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// RequireAuth — проверяет, что пользователь залогинен (есть user_id в сессии).
// Если нет — редиректит на /login.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)

		if uid, ok := sess.Get("user_id").(uint); !ok || uid == 0 {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole — пускает только пользователей с одной из указанных ролей.
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		roleStr, _ := sess.Get("role").(string)
		current := models.UserRole(roleStr)

		allowed := false
		for _, r := range roles {
			if r == current {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

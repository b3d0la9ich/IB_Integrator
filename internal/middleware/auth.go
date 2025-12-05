package middleware

import (
	"net/http"

	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		userID := sess.Get("user_id")
		if userID == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	roleSet := map[models.UserRole]struct{}{}
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		sess := sessions.Default(c)
		roleVal := sess.Get("role")
		roleStr, ok := roleVal.(string)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		role := models.UserRole(roleStr)

		if _, ok := roleSet[role]; !ok {
			c.String(http.StatusForbidden, "access denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

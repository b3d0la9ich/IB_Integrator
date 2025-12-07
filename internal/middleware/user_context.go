package middleware

import (
    "ib-integrator/internal/database"
    "ib-integrator/internal/models"

    "github.com/gin-contrib/sessions"
    "github.com/gin-gonic/gin"
)

func InjectUser() gin.HandlerFunc {
    return func(c *gin.Context) {
        sess := sessions.Default(c)

        if uidRaw := sess.Get("user_id"); uidRaw != nil {
            if uid, ok := uidRaw.(uint); ok && uid > 0 {
                var user models.User
                if err := database.DB.First(&user, uid).Error; err == nil {
                    c.Set("CurrentUser", user)
                }
            }
        }

        c.Next()
    }
}

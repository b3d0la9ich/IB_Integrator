package handlers

import (
	"ib-integrator/internal/models"

	"github.com/gin-gonic/gin"
)

// render — обёртка над c.HTML, которая во все шаблоны прокидывает CurrentUser.
func render(c *gin.Context, status int, tmpl string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	// Пытаемся достать пользователя, которого положил middleware.InjectUser
	if uVal, ok := c.Get("CurrentUser"); ok {
		switch u := uVal.(type) {
		case models.User:
			data["CurrentUser"] = u
			data["CurrentUsername"] = u.Username
			data["CurrentUserRole"] = u.Role
		case *models.User:
			data["CurrentUser"] = u
			data["CurrentUsername"] = u.Username
			data["CurrentUserRole"] = u.Role
		}
	}

	// обычный рендер
	c.HTML(status, tmpl, data)
}

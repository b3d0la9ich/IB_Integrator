package handlers

import (
	"net/http"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func ListAuditLogs(c *gin.Context) {
	// достаём роль из сессии
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)

	// можно сразу ограничить доступ
	if role != models.RoleAdmin && role != models.RoleViewer {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	var logs []models.AuditLog
	database.DB.
		Preload("User").
		Order("created_at desc").
		Limit(200).
		Find(&logs)

	render(c, http.StatusOK, "audit_list.html", gin.H{
		"logs": logs,
		"role": roleStr,                    // <- нужно для {{ .role }} в шаблоне
		"IsAdmin": role == models.RoleAdmin, // если потом захочешь использовать
	})
}

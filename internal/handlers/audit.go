package handlers

import (
	"net/http"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-gonic/gin"
)

func ListAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	database.DB.
		Preload("User").
		Order("created_at desc").
		Limit(200).
		Find(&logs)

	c.HTML(http.StatusOK, "audit_list.html", gin.H{
		"logs": logs,
	})
}

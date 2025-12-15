package handlers

import (
	"net/http"
	"strconv"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-gonic/gin"
)

func ShowClientDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		c.String(http.StatusBadRequest, "Некорректный ID клиента")
		return
	}

	var client models.Client
	// Грузим клиента сразу с объектами защиты и проектами
	if err := database.DB.
		Preload("Assets").
		First(&client, id).Error; err != nil {
		c.String(http.StatusNotFound, "Клиент не найден")
		return
	}

	render(c, http.StatusOK, "client_detail.html", gin.H{
		"client": client,
	})
}

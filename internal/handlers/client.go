package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// helper: кто может управлять клиентами (admin + sales)
func isAdmin(c *gin.Context) bool {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)
	return role == models.RoleAdmin || role == models.RoleSales
}

//
// СПИСОК / СОЗДАНИЕ
//

func ListClients(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)

	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	render(c, http.StatusOK, "clients_list.html", gin.H{
		"clients": clients,
		"IsAdmin": role == models.RoleAdmin, // именно "настоящий" админ
	})
}

func ShowNewClient(c *gin.Context) {
	if !isAdmin(c) {
		c.String(http.StatusForbidden, "Недостаточно прав")
		return
	}

	render(c, http.StatusOK, "clients_new.html", gin.H{
		"error": "",
	})
}

func CreateClient(c *gin.Context) {
	if !isAdmin(c) {
		c.String(http.StatusForbidden, "Недостаточно прав")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	orgType := strings.TrimSpace(c.PostForm("org_type"))
	inn := strings.TrimSpace(c.PostForm("inn"))
	industry := strings.TrimSpace(c.PostForm("industry"))
	contactEmail := strings.TrimSpace(c.PostForm("contact_email"))
	contactPhone := strings.TrimSpace(c.PostForm("contact_phone"))
	notes := strings.TrimSpace(c.PostForm("notes"))

	if len(name) < 3 {
		renderClientError(c, "Название организации должно быть не короче 3 символов")
		return
	}

	client := models.Client{
		Name:         name,
		OrgType:      orgType,
		INN:          inn,
		Industry:     industry,
		ContactEmail: contactEmail,
		ContactPhone: contactPhone,
		Notes:        notes,
	}

	if err := database.DB.Create(&client).Error; err != nil {
		renderClientError(c, "Ошибка сохранения клиента в БД")
		return
	}

	c.Redirect(http.StatusFound, "/clients")
}

// форма редактирования
func ShowEditClient(c *gin.Context) {
	if !isAdmin(c) {
		c.String(http.StatusForbidden, "Недостаточно прав")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID клиента")
		return
	}

	var client models.Client
	if err := database.DB.First(&client, id).Error; err != nil {
		c.String(http.StatusNotFound, "Клиент не найден")
		return
	}

	render(c, http.StatusOK, "clients_edit.html", gin.H{
		"client": client,
		"error":  "",
	})
}

// сохранение изменений
func UpdateClient(c *gin.Context) {
	if !isAdmin(c) {
		c.String(http.StatusForbidden, "Недостаточно прав")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID клиента")
		return
	}

	var client models.Client
	if err := database.DB.First(&client, id).Error; err != nil {
		c.String(http.StatusNotFound, "Клиент не найден")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	orgType := strings.TrimSpace(c.PostForm("org_type"))
	inn := strings.TrimSpace(c.PostForm("inn"))
	industry := strings.TrimSpace(c.PostForm("industry"))
	contactEmail := strings.TrimSpace(c.PostForm("contact_email"))
	contactPhone := strings.TrimSpace(c.PostForm("contact_phone"))
	notes := strings.TrimSpace(c.PostForm("notes"))

	if len(name) < 3 {
		render(c, http.StatusBadRequest, "clients_edit.html", gin.H{
			"client": client,
			"error":  "Название организации должно быть не короче 3 символов",
		})
		return
	}

	client.Name = name
	client.OrgType = orgType
	client.INN = inn
	client.Industry = industry
	client.ContactEmail = contactEmail
	client.ContactPhone = contactPhone
	client.Notes = notes

	if err := database.DB.Save(&client).Error; err != nil {
		render(c, http.StatusInternalServerError, "clients_edit.html", gin.H{
			"client": client,
			"error":  "Ошибка сохранения клиента",
		})
		return
	}

	c.Redirect(http.StatusFound, "/clients/"+idStr)
}

func renderClientError(c *gin.Context, msg string) {
	render(c, http.StatusBadRequest, "clients_new.html", gin.H{
		"error": msg,
	})
}

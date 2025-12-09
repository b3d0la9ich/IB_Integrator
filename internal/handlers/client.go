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

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ ИНН ---
	if inn != "" {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("inn = ?", inn).
			Count(&count)

		if count > 0 {
			renderClientError(c, "Клиент с таким ИНН уже существует")
			return
		}
	}

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ ИМЕНИ ---
	if name != "" {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("LOWER(name) = LOWER(?)", name).
			Count(&count)

		if count > 0 {
			renderClientError(c, "Клиент с таким названием уже существует")
			return
		}
	}

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ EMAIL ---
	if contactEmail != "" {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("LOWER(contact_email) = LOWER(?)", contactEmail).
			Count(&count)

		if count > 0 {
			renderClientError(c, "Клиент с таким e-mail уже существует")
			return
		}
	}

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ ТЕЛЕФОНА ---
	if contactPhone != "" {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("contact_phone = ?", contactPhone).
			Count(&count)

		if count > 0 {
			renderClientError(c, "Клиент с таким номером телефона уже существует")
			return
		}
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

	// --- АУДИТ: создание клиента ---
	sess := sessions.Default(c)
	if v := sess.Get("user_id"); v != nil {
		if uid, ok := v.(uint); ok {
			database.CreateAuditLog(uid, "client", client.ID, "create", "Создан клиент: "+client.Name)
		}
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

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ ИНН (кроме текущего клиента) ---
	if inn != "" && inn != client.INN {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("inn = ? AND id <> ?", inn, client.ID).
			Count(&count)

		if count > 0 {
			render(c, http.StatusBadRequest, "clients_edit.html", gin.H{
				"client": client,
				"error":  "Клиент с таким ИНН уже существует",
			})
			return
		}
	}

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ ИМЕНИ ---
	if name != "" && name != client.Name {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("LOWER(name) = LOWER(?) AND id <> ?", name, client.ID).
			Count(&count)

		if count > 0 {
			render(c, http.StatusBadRequest, "clients_edit.html", gin.H{
				"client": client,
				"error":  "Клиент с таким названием уже существует",
			})
			return
		}
	}

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ EMAIL ---
	if contactEmail != "" && !strings.EqualFold(contactEmail, client.ContactEmail) {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("LOWER(contact_email) = LOWER(?) AND id <> ?", contactEmail, client.ID).
			Count(&count)

		if count > 0 {
			render(c, http.StatusBadRequest, "clients_edit.html", gin.H{
				"client": client,
				"error":  "Клиент с таким e-mail уже существует",
			})
			return
		}
	}

	// --- ПРОВЕРКА УНИКАЛЬНОСТИ ТЕЛЕФОНА ---
	if contactPhone != "" && contactPhone != client.ContactPhone {
		var count int64
		database.DB.Model(&models.Client{}).
			Where("contact_phone = ? AND id <> ?", contactPhone, client.ID).
			Count(&count)

		if count > 0 {
			render(c, http.StatusBadRequest, "clients_edit.html", gin.H{
				"client": client,
				"error":  "Клиент с таким номером телефона уже существует",
			})
			return
		}
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

	// --- АУДИТ: изменение клиента ---
	sess := sessions.Default(c)
	if v := sess.Get("user_id"); v != nil {
		if uid, ok := v.(uint); ok {
			database.CreateAuditLog(uid, "client", client.ID, "update", "Изменён клиент: "+client.Name)
		}
	}

	c.Redirect(http.StatusFound, "/clients/"+idStr)
}

func renderClientError(c *gin.Context, msg string) {
	render(c, http.StatusBadRequest, "clients_new.html", gin.H{
		"error": msg,
	})
}

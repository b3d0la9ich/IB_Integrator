package handlers

import (
	"net/http"
	"strings"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// СПИСОК ОБЪЕКТОВ ЗАЩИТЫ

func ListAssets(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)

	var assets []models.Asset
	database.DB.Preload("Client").Order("client_id asc, name asc").Find(&assets)

	render(c, http.StatusOK, "assets_list.html", gin.H{
		"assets": assets,
		"role":   roleStr,
	})
}

// СОЗДАНИЕ НОВОГО ОБЪЕКТА

func ShowNewAsset(c *gin.Context) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	render(c, http.StatusOK, "assets_new.html", gin.H{
		"clients": clients,
		"error":   "",
	})
}

func CreateAsset(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	clientIDStr := c.PostForm("client_id")
	aTypeStr := strings.TrimSpace(c.PostForm("asset_type"))
	category := strings.TrimSpace(c.PostForm("category"))
	description := strings.TrimSpace(c.PostForm("description"))

	if len(name) < 3 {
		renderAssetError(c, "Название объекта защиты должно быть не короче 3 символов")
		return
	}

	if aTypeStr == "" {
		renderAssetError(c, "Укажите тип объекта защиты")
		return
	}

	var client models.Client
	if err := database.DB.First(&client, clientIDStr).Error; err != nil {
		renderAssetError(c, "Клиент не найден")
		return
	}

	// Для ИСПДн/ГИС — требуем указать класс/уровень защищённости
	upperType := strings.ToUpper(aTypeStr)
	if (strings.Contains(upperType, "ИСПД") || strings.Contains(upperType, "ГИС")) && category == "" {
		renderAssetError(c, "Для ИСПДн/ГИС необходимо указать класс/уровень защищённости")
		return
	}

	asset := models.Asset{
		ClientID:    client.ID,
		Name:        name,
		AssetType:   models.AssetType(aTypeStr),
		Category:    category,
		Description: description,
	}

	if err := database.DB.Create(&asset).Error; err != nil {
		renderAssetError(c, "Ошибка сохранения объекта защиты в БД")
		return
	}

	sess := sessions.Default(c)
	if uid, ok := sess.Get("user_id").(uint); ok {
		database.CreateAuditLog(uid, "asset", asset.ID, "create", "Создан объект защиты: "+asset.Name)
	}

	c.Redirect(http.StatusFound, "/assets")
}

func renderAssetError(c *gin.Context, msg string) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	render(c, http.StatusBadRequest, "assets_new.html", gin.H{
		"error":   msg,
		"clients": clients,
	})
}

// РЕДАКТИРОВАНИЕ ОБЪЕКТА

func ShowEditAsset(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)
	if role != models.RoleAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	id := c.Param("id")

	var asset models.Asset
	if err := database.DB.Preload("Client").First(&asset, id).Error; err != nil {
		c.String(http.StatusNotFound, "Объект защиты не найден")
		return
	}

	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	render(c, http.StatusOK, "assets_edit.html", gin.H{
		"asset":   asset,
		"clients": clients,
		"error":   "",
	})
}

func UpdateAsset(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)
	if role != models.RoleAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	id := c.Param("id")

	var asset models.Asset
	if err := database.DB.First(&asset, id).Error; err != nil {
		c.String(http.StatusNotFound, "Объект защиты не найден")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	clientIDStr := c.PostForm("client_id")
	aTypeStr := strings.TrimSpace(c.PostForm("asset_type"))
	category := strings.TrimSpace(c.PostForm("category"))
	description := strings.TrimSpace(c.PostForm("description"))

	if len(name) < 3 {
		renderAssetEditError(c, asset, "Название объекта защиты должно быть не короче 3 символов")
		return
	}

	if aTypeStr == "" {
		renderAssetEditError(c, asset, "Укажите тип объекта защиты")
		return
	}

	var client models.Client
	if err := database.DB.First(&client, clientIDStr).Error; err != nil {
		renderAssetEditError(c, asset, "Клиент не найден")
		return
	}

	upperType := strings.ToUpper(aTypeStr)
	if (strings.Contains(upperType, "ИСПД") || strings.Contains(upperType, "ГИС")) && category == "" {
		renderAssetEditError(c, asset, "Для ИСПДн/ГИС необходимо указать класс/уровень защищённости")
		return
	}

	asset.ClientID = client.ID
	asset.Name = name
	asset.AssetType = models.AssetType(aTypeStr)
	asset.Category = category
	asset.Description = description

	if err := database.DB.Save(&asset).Error; err != nil {
		renderAssetEditError(c, asset, "Ошибка сохранения объекта защиты в БД")
		return
	}

	if uid, ok := sess.Get("user_id").(uint); ok {
		database.CreateAuditLog(uid, "asset", asset.ID, "update", "Изменён объект защиты: "+asset.Name)
	}

	c.Redirect(http.StatusFound, "/assets")
}

func renderAssetEditError(c *gin.Context, asset models.Asset, msg string) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	render(c, http.StatusBadRequest, "assets_edit.html", gin.H{
		"error":   msg,
		"asset":   asset,
		"clients": clients,
	})
}

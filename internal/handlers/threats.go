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

// ====== ДОСТУП К РИСКАМ (УГРОЗАМ / МЕРАМ) ======

func requireRiskEditor(c *gin.Context) (models.UserRole, bool) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)

	if role != models.RoleAdmin && role != models.RoleEngineer {
		c.AbortWithStatus(http.StatusForbidden)
		return role, false
	}
	return role, true
}

// ====== КАТАЛОГ УГРОЗ И МЕР ======

func ListThreatsAndMeasures(c *gin.Context) {
	role, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	var threats []models.Threat
	var measures []models.ControlMeasure
	var links []models.ThreatMeasure

	database.DB.Order("code asc").Find(&threats)
	database.DB.Order("code asc").Find(&measures)
	database.DB.Preload("Measure").Order("threat_id asc, measure_id asc").Find(&links)

	// строим карту: threatID -> []ControlMeasure
	rec := make(map[uint][]models.ControlMeasure)
	for _, l := range links {
		if l.Measure.ID == 0 {
			continue
		}
		rec[l.ThreatID] = append(rec[l.ThreatID], l.Measure)
	}

	render(c, http.StatusOK, "threats_list.html", gin.H{
		"role":        string(role),
		"threats":     threats,
		"measures":    measures,
		"RecMeasures": rec,
	})
}


// --- Угрозы: создание

func ShowNewThreat(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	render(c, http.StatusOK, "threats_new.html", gin.H{
		"error": "",
	})
}

func CreateThreat(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	code := strings.TrimSpace(c.PostForm("code"))
	name := strings.TrimSpace(c.PostForm("name"))
	category := strings.TrimSpace(c.PostForm("category"))
	desc := strings.TrimSpace(c.PostForm("description"))

	if len(name) < 3 {
		render(c, http.StatusBadRequest, "threats_new.html", gin.H{
			"error": "Название угрозы должно быть не короче 3 символов",
		})
		return
	}

	th := models.Threat{
		Code:        code,
		Name:        name,
		Category:    category,
		Description: desc,
	}

	if err := database.DB.Create(&th).Error; err != nil {
		render(c, http.StatusBadRequest, "threats_new.html", gin.H{
			"error": "Ошибка сохранения угрозы в БД",
		})
		return
	}

	c.Redirect(http.StatusFound, "/threats")
}

// --- Меры: создание

func ShowNewMeasure(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	render(c, http.StatusOK, "measures_new.html", gin.H{
		"error": "",
	})
}

func CreateMeasure(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	code := strings.TrimSpace(c.PostForm("code"))
	name := strings.TrimSpace(c.PostForm("name"))
	standard := strings.TrimSpace(c.PostForm("standard"))
	desc := strings.TrimSpace(c.PostForm("description"))

	if len(name) < 3 {
		render(c, http.StatusBadRequest, "measures_new.html", gin.H{
			"error": "Название меры защиты должно быть не короче 3 символов",
		})
		return
	}

	m := models.ControlMeasure{
		Code:        code,
		Name:        name,
		Standard:    standard,
		Description: desc,
	}

	if err := database.DB.Create(&m).Error; err != nil {
		render(c, http.StatusBadRequest, "measures_new.html", gin.H{
			"error": "Ошибка сохранения меры защиты в БД",
		})
		return
	}

	c.Redirect(http.StatusFound, "/threats")
}

// ====== УГРОЗЫ КОНКРЕТНОГО ОБЪЕКТА ЗАЩИТЫ ======

func ShowAssetThreats(c *gin.Context) {
	role, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	assetID, err := strconv.Atoi(idStr)
	if err != nil || assetID <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID объекта защиты")
		return
	}

	var asset models.Asset
	if err := database.DB.Preload("Client").First(&asset, assetID).Error; err != nil {
		c.String(http.StatusNotFound, "Объект защиты не найден")
		return
	}

	var links []models.AssetThreat
	database.DB.
		Preload("Threat").
		Where("asset_id = ?", asset.ID).
		Order("id asc").
		Find(&links)

	// Каталог угроз, которые ещё не привязаны к этому объекту
	var usedIDs []uint
	for _, l := range links {
		usedIDs = append(usedIDs, l.ThreatID)
	}

	thQuery := database.DB.Order("code asc")
	if len(usedIDs) > 0 {
		thQuery = thQuery.Where("id NOT IN ?", usedIDs)
	}

	var threats []models.Threat
	thQuery.Find(&threats)

	render(c, http.StatusOK, "asset_threats.html", gin.H{
		"role":   string(role),
		"asset":  asset,
		"links":  links,
		"threats": threats,
	})
}

func AddAssetThreat(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	assetID, err := strconv.Atoi(idStr)
	if err != nil || assetID <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID объекта защиты")
		return
	}

	threatIDStr := c.PostForm("threat_id")
	risk := strings.TrimSpace(c.PostForm("risk_level"))
	notes := strings.TrimSpace(c.PostForm("notes"))

	tid, err := strconv.Atoi(threatIDStr)
	if err != nil || tid <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID угрозы")
		return
	}

	// Проверка уровня риска
	switch risk {
	case "low", "medium", "high":
	default:
		c.String(http.StatusBadRequest, "Некорректный уровень риска")
		return
	}

	// Проверка отсутствия дубликата
	var count int64
	database.DB.Model(&models.AssetThreat{}).
		Where("asset_id = ? AND threat_id = ?", assetID, tid).
		Count(&count)
	if count > 0 {
		c.String(http.StatusBadRequest, "Эта угроза уже привязана к объекту")
		return
	}

	link := models.AssetThreat{
		AssetID:   uint(assetID),
		ThreatID:  uint(tid),
		RiskLevel: risk,
		Notes:     notes,
	}

	if err := database.DB.Create(&link).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка сохранения угрозы для объекта")
		return
	}

	c.Redirect(http.StatusFound, "/assets/"+idStr+"/threats")
}

func DeleteAssetThreat(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	assetIDStr := c.Param("id")
	linkIDStr := c.Param("link_id")

	assetID, err1 := strconv.Atoi(assetIDStr)
	linkID, err2 := strconv.Atoi(linkIDStr)

	if err1 != nil || err2 != nil || assetID <= 0 || linkID <= 0 {
		c.String(http.StatusBadRequest, "Некорректные параметры")
		return
	}

	if err := database.DB.Where("id = ? AND asset_id = ?", linkID, assetID).
		Delete(&models.AssetThreat{}).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка удаления связи угрозы")
		return
	}

	c.Redirect(http.StatusFound, "/assets/"+assetIDStr+"/threats")
}

// ====== МЕРЫ ЗАЩИТЫ КОНКРЕТНОГО ПРОЕКТА ======

func ShowProjectMeasures(c *gin.Context) {
	role, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	pid, err := strconv.Atoi(idStr)
	if err != nil || pid <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID проекта")
		return
	}

	var project models.Project
	if err := database.DB.Preload("Client").Preload("Asset").First(&project, pid).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	var links []models.ProjectMeasure
	database.DB.
		Preload("Measure").
		Where("project_id = ?", project.ID).
		Order("id asc").
		Find(&links)

	// Каталог мер, которые ещё не привязаны к проекту
	var usedIDs []uint
	for _, l := range links {
		usedIDs = append(usedIDs, l.MeasureID)
	}

	mQuery := database.DB.Order("code asc")
	if len(usedIDs) > 0 {
		mQuery = mQuery.Where("id NOT IN ?", usedIDs)
	}

	var measures []models.ControlMeasure
	mQuery.Find(&measures)

	render(c, http.StatusOK, "project_measures.html", gin.H{
		"role":     string(role),
		"project":  project,
		"links":    links,
		"measures": measures,
	})
}

func AddProjectMeasure(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	pid, err := strconv.Atoi(idStr)
	if err != nil || pid <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID проекта")
		return
	}

	measureIDStr := c.PostForm("measure_id")
	status := strings.TrimSpace(c.PostForm("status"))
	notes := strings.TrimSpace(c.PostForm("notes"))

	mid, err := strconv.Atoi(measureIDStr)
	if err != nil || mid <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID меры")
		return
	}

	switch status {
	case "planned", "in_progress", "done":
	default:
		c.String(http.StatusBadRequest, "Некорректный статус реализации меры")
		return
	}

	var count int64
	database.DB.Model(&models.ProjectMeasure{}).
		Where("project_id = ? AND measure_id = ?", pid, mid).
		Count(&count)
	if count > 0 {
		c.String(http.StatusBadRequest, "Эта мера уже привязана к проекту")
		return
	}

	link := models.ProjectMeasure{
		ProjectID: uint(pid),
		MeasureID: uint(mid),
		Status:    status,
		Notes:     notes,
	}

	if err := database.DB.Create(&link).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка сохранения меры для проекта")
		return
	}

	c.Redirect(http.StatusFound, "/projects/"+idStr+"/measures")
}

func DeleteProjectMeasure(c *gin.Context) {
	_, ok := requireRiskEditor(c)
	if !ok {
		return
	}

	pidStr := c.Param("id")
	linkIDStr := c.Param("link_id")

	pid, err1 := strconv.Atoi(pidStr)
	linkID, err2 := strconv.Atoi(linkIDStr)

	if err1 != nil || err2 != nil || pid <= 0 || linkID <= 0 {
		c.String(http.StatusBadRequest, "Некорректные параметры")
		return
	}

	if err := database.DB.Where("id = ? AND project_id = ?", linkID, pid).
		Delete(&models.ProjectMeasure{}).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка удаления связи меры")
		return
	}

	c.Redirect(http.StatusFound, "/projects/"+pidStr+"/measures")
}

package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

//
// СПИСОК ПРОЕКТОВ
//

// Список проектов + фильтры
func ListProjects(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)

	clientIDStr := c.Query("client_id")
	typeStr := c.Query("type")
	statusStr := c.Query("status")

	dbq := database.DB.Preload("Client").Preload("Asset").Order("created_at desc")

	if clientIDStr != "" {
		if cid, err := strconv.Atoi(clientIDStr); err == nil && cid > 0 {
			dbq = dbq.Where("client_id = ?", cid)
		}
	}

	if typeStr != "" {
		dbq = dbq.Where("type = ?", typeStr)
	}

	if statusStr != "" {
		dbq = dbq.Where("status = ?", statusStr)
	}

	var projects []models.Project
	if err := dbq.Find(&projects).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка загрузки проектов")
		return
	}

	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	render(c, http.StatusOK, "projects_list.html", gin.H{
		"projects":       projects,
		"clients":        clients,
		"FilterClientID": clientIDStr,
		"FilterType":     typeStr,
		"FilterStatus":   statusStr,

		"IsAdmin":    role == models.RoleAdmin,
		"IsSales":    role == models.RoleSales,
		"IsEngineer": role == models.RoleEngineer,
	})
}

//
// СОЗДАНИЕ ПРОЕКТА
//

func ShowNewProject(c *gin.Context) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	var assets []models.Asset
	database.DB.Order("name asc").Find(&assets)

	var engineers []models.User
	database.DB.Where("role = ?", models.RoleEngineer).Order("username asc").Find(&engineers)

	render(c, http.StatusOK, "projects_new.html", gin.H{
		"clients":   clients,
		"assets":    assets,
		"engineers": engineers,
		"error":     "",
	})
}

func CreateProject(c *gin.Context) {
	title := strings.TrimSpace(c.PostForm("name"))
	clientIDStr := c.PostForm("client_id")
	assetIDStr := c.PostForm("asset_id")
	projectTypeStr := c.PostForm("project_type")
	engineerIDStr := c.PostForm("engineer_id")
	desc := strings.TrimSpace(c.PostForm("description"))
	plannedStartStr := c.PostForm("planned_start")
	plannedEndStr := c.PostForm("planned_end")

	if len(title) < 3 {
		renderProjectError(c, "Название проекта должно быть не короче 3 символов")
		return
	}

	cid, err := strconv.Atoi(clientIDStr)
	if err != nil || cid <= 0 {
		renderProjectError(c, "Выберите клиента")
		return
	}

	ptype := models.ProjectType(projectTypeStr)
	switch ptype {
	case models.ProjectAudit,
		models.ProjectSZIDeploy,
		models.ProjectPentest,
		models.ProjectMaintenance,
		models.ProjectCompliance:
	default:
		renderProjectError(c, "Неверный тип проекта")
		return
	}

	var client models.Client
	if err := database.DB.First(&client, cid).Error; err != nil {
		renderProjectError(c, "Клиент не найден")
		return
	}

	var assetID uint
	if assetIDStr != "" {
		if aid, err := strconv.Atoi(assetIDStr); err == nil {
			assetID = uint(aid)
		}
	}

	var engineerID uint
	if engineerIDStr != "" {
		if eid, err := strconv.Atoi(engineerIDStr); err == nil {
			engineerID = uint(eid)
		}
	}

	var plannedStart *time.Time
	if plannedStartStr != "" {
		if t, err := time.Parse("2006-01-02", plannedStartStr); err == nil {
			plannedStart = &t
		}
	}

	var plannedEnd *time.Time
	if plannedEndStr != "" {
		if t, err := time.Parse("2006-01-02", plannedEndStr); err == nil {
			plannedEnd = &t
		}
	}

	sess := sessions.Default(c)
	var salesID uint
	if uid, ok := sess.Get("user_id").(uint); ok {
		salesID = uid
	}

	project := models.Project{
		ClientID:     client.ID,
		AssetID:      assetID,
		Title:        title,
		Type:         ptype,
		Status:       models.StatusPlanned,
		Description:  desc,
		PlannedStart: plannedStart,
		PlannedEnd:   plannedEnd,
		SalesID:      salesID,
		EngineerID:   engineerID,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		renderProjectError(c, "Ошибка сохранения проекта")
		return
	}

	if salesID != 0 {
		database.CreateAuditLog(salesID, "project", project.ID, "create", "Создан проект: "+project.Title)
	}

	c.Redirect(http.StatusFound, "/projects")
}

func renderProjectError(c *gin.Context, msg string) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	var assets []models.Asset
	database.DB.Order("name asc").Find(&assets)

	var engineers []models.User
	database.DB.Where("role = ?", models.RoleEngineer).Order("username asc").Find(&engineers)

	render(c, http.StatusBadRequest, "projects_new.html", gin.H{
		"error":     msg,
		"clients":   clients,
		"assets":    assets,
		"engineers": engineers,
	})
}

//
// СМЕНА СТАТУСА
//

func ChangeProjectStatus(c *gin.Context) {
	idStr := c.Param("id")
	statusStr := c.PostForm("status")

	pid, err := strconv.Atoi(idStr)
	if err != nil || pid <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID проекта")
		return
	}

	var project models.Project
	if err := database.DB.First(&project, pid).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	newStatus := models.ProjectStatus(statusStr)

	switch newStatus {
	case models.StatusPlanned,
		models.StatusInProgress,
		models.StatusOnApproval,
		models.StatusFinished,
		models.StatusCancelled:
	default:
		c.String(http.StatusBadRequest, "Некорректный статус")
		return
	}

	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)

	if !canChangeProjectStatus(role, project.Status, newStatus) {
		c.String(http.StatusForbidden, "Недостаточно прав")
		return
	}

	if newStatus == models.StatusFinished {
		now := time.Now()
		project.ActualEnd = &now
	}

	project.Status = newStatus

	if err := database.DB.Save(&project).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка обновления статуса")
		return
	}

	if uid, ok := sess.Get("user_id").(uint); ok {
		database.CreateAuditLog(uid, "project", project.ID, "status_change",
			"Статус изменён на: "+string(newStatus))
	}

	c.Redirect(http.StatusFound, "/projects")
}

// логика ролей
func canChangeProjectStatus(role models.UserRole, current, next models.ProjectStatus) bool {
	if current == next {
		return false
	}

	switch role {

	case models.RoleAdmin:
		return true

	case models.RoleSales:
		switch current {
		case models.StatusPlanned:
			return next == models.StatusInProgress || next == models.StatusCancelled
		case models.StatusOnApproval:
			return next == models.StatusInProgress || next == models.StatusCancelled
		}
		return false

	case models.RoleEngineer:
		return current == models.StatusInProgress && next == models.StatusOnApproval

	default:
		return false
	}
}

//
// РЕДАКТИРОВАНИЕ ПРОЕКТА
//

func ShowEditProject(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	if models.UserRole(roleStr) != models.RoleAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	id := c.Param("id")
	var project models.Project
	if err := database.DB.Preload("Client").Preload("Asset").First(&project, id).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	var assets []models.Asset
	database.DB.Order("name asc").Find(&assets)

	var engineers []models.User
	database.DB.Where("role = ?", models.RoleEngineer).Order("username asc").Find(&engineers)

	render(c, http.StatusOK, "projects_edit.html", gin.H{
		"project":   project,
		"clients":   clients,
		"assets":    assets,
		"engineers": engineers,
		"error":     "",
	})
}

func UpdateProject(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	if models.UserRole(roleStr) != models.RoleAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	id := c.Param("id")

	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	title := strings.TrimSpace(c.PostForm("name"))
	clientIDStr := c.PostForm("client_id")
	assetIDStr := c.PostForm("asset_id")
	projectType := c.PostForm("project_type")
	engineerIDStr := c.PostForm("engineer_id")
	plannedStartStr := c.PostForm("planned_start")
	plannedEndStr := c.PostForm("planned_end")
	description := strings.TrimSpace(c.PostForm("description"))

	if len(title) < 3 {
		renderProjectEditError(c, project, "Название слишком короткое")
		return
	}

	// клиент обязателен
	var client models.Client
	if err := database.DB.First(&client, clientIDStr).Error; err != nil {
		renderProjectEditError(c, project, "Клиент не найден")
		return
	}

	// объект защиты
	var assetID uint
	if assetIDStr != "" {
		var asset models.Asset
		if err := database.DB.First(&asset, assetIDStr).Error; err != nil {
			renderProjectEditError(c, project, "Объект защиты не найден")
			return
		}
		assetID = asset.ID
	}

	// инженер
	var engineerID uint
	if engineerIDStr != "" {
		var engineer models.User
		if err := database.DB.First(&engineer, engineerIDStr).Error; err != nil {
			renderProjectEditError(c, project, "Инженер не найден")
			return
		}
		engineerID = engineer.ID
	}

	var plannedStart *time.Time
	if plannedStartStr != "" {
		t, err := time.Parse("2006-01-02", plannedStartStr)
		if err != nil {
			renderProjectEditError(c, project, "Неверная дата начала")
			return
		}
		plannedStart = &t
	}

	var plannedEnd *time.Time
	if plannedEndStr != "" {
		t, err := time.Parse("2006-01-02", plannedEndStr)
		if err != nil {
			renderProjectEditError(c, project, "Неверная дата окончания")
			return
		}
		plannedEnd = &t
	}

	project.Title = title
	project.ClientID = client.ID
	project.Description = description
	project.Type = models.ProjectType(projectType)
	project.PlannedStart = plannedStart
	project.PlannedEnd = plannedEnd
	project.AssetID = assetID
	project.EngineerID = engineerID

	if err := database.DB.Save(&project).Error; err != nil {
		renderProjectEditError(c, project, "Ошибка сохранения проекта")
		return
	}

	if uid, ok := sess.Get("user_id").(uint); ok {
		database.CreateAuditLog(uid, "project", project.ID, "update", "Проект обновлён: "+project.Title)
	}

	c.Redirect(http.StatusFound, "/projects")
}

func renderProjectEditError(c *gin.Context, project models.Project, msg string) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	var assets []models.Asset
	database.DB.Order("name asc").Find(&assets)

	var engineers []models.User
	database.DB.Where("role = ?", models.RoleEngineer).Order("username asc").Find(&engineers)

	render(c, http.StatusBadRequest, "projects_edit.html", gin.H{
		"error":     msg,
		"project":   project,
		"clients":   clients,
		"assets":    assets,
		"engineers": engineers,
	})
}

//
// УДАЛЕНИЕ ПРОЕКТА
//

func DeleteProject(c *gin.Context) {
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	if models.UserRole(roleStr) != models.RoleAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID")
		return
	}

	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	if err := database.DB.Delete(&project).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка удаления")
		return
	}

	if uid, ok := sess.Get("user_id").(uint); ok {
		database.CreateAuditLog(uid, "project", project.ID, "delete", "Удалён проект: "+project.Title)
	}

	c.Redirect(http.StatusFound, "/projects")
}

//
// ИСТОРИЯ ПРОЕКТА
//

func ShowProjectHistory(c *gin.Context) {
	idStr := c.Param("id")
	pid, err := strconv.Atoi(idStr)
	if err != nil || pid <= 0 {
		c.String(http.StatusBadRequest, "Некорректный ID")
		return
	}

	var project models.Project
	if err := database.DB.First(&project, pid).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	var logs []models.AuditLog
	database.DB.Where("entity = ? AND entity_id = ?", "project", pid).
		Preload("User").
		Order("created_at asc").
		Find(&logs)

	render(c, http.StatusOK, "project_history.html", gin.H{
		"project": project,
		"logs":    logs,
	})
}

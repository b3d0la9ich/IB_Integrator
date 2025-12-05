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

// Список проектов с фильтрами
func ListProjects(c *gin.Context) {
	// достаём роль из сессии
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

	c.HTML(http.StatusOK, "projects_list.html", gin.H{
		"projects":       projects,
		"clients":        clients,
		"FilterClientID": clientIDStr,
		"FilterType":     typeStr,
		"FilterStatus":   statusStr,

		// флаги для шаблона
		"IsAdmin":    role == models.RoleAdmin,
		"IsSales":    role == models.RoleSales,
		"IsEngineer": role == models.RoleEngineer,
	})
}

//
// ФОРМА СОЗДАНИЯ ПРОЕКТА
//

func ShowNewProject(c *gin.Context) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	var assets []models.Asset
	database.DB.Order("name asc").Find(&assets)

	// инженеры по ИБ (роль engineer)
	var engineers []models.User
	database.DB.Where("role = ?", models.RoleEngineer).Order("username asc").Find(&engineers)

	c.HTML(http.StatusOK, "projects_new.html", gin.H{
		"clients":   clients,
		"assets":    assets,
		"engineers": engineers,
		"error":     "",
	})
}

//
// СОЗДАНИЕ ПРОЕКТА
//

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

	// клиент обязателен
	cid, err := strconv.Atoi(clientIDStr)
	if err != nil || cid <= 0 {
		renderProjectError(c, "Выберите клиента")
		return
	}

	// тип проверяем по enum
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
		if aid, err := strconv.Atoi(assetIDStr); err == nil && aid > 0 {
			assetID = uint(aid)
		}
	}

	var engineerID uint
	if engineerIDStr != "" {
		if eid, err := strconv.Atoi(engineerIDStr); err == nil && eid > 0 {
			engineerID = uint(eid)
		}
	}

	var plannedStartPtr *time.Time
	if plannedStartStr != "" {
		if t, err := time.Parse("2006-01-02", plannedStartStr); err == nil {
			plannedStartPtr = &t
		}
	}

	var plannedEndPtr *time.Time
	if plannedEndStr != "" {
		if t, err := time.Parse("2006-01-02", plannedEndStr); err == nil {
			plannedEndPtr = &t
		}
	}

	// sales — это текущий пользователь (admin/sales)
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
		PlannedStart: plannedStartPtr,
		PlannedEnd:   plannedEndPtr,
		SalesID:      salesID,
		EngineerID:   engineerID,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		renderProjectError(c, "Ошибка сохранения проекта в БД")
		return
	}

	if salesID != 0 {
		database.CreateAuditLog(salesID, "project", project.ID, "create", "Создан проект: "+project.Title)
	}

	c.Redirect(http.StatusFound, "/projects")
}

//
// СМЕНА СТАТУСА ПРОЕКТА (вариант B)
//

func ChangeProjectStatus(c *gin.Context) {
	idStr := c.Param("id")
	newStatusStr := c.PostForm("status")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.String(http.StatusBadRequest, "Некорректный идентификатор проекта")
		return
	}

	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		c.String(http.StatusNotFound, "Проект не найден")
		return
	}

	status := models.ProjectStatus(newStatusStr)
	switch status {
	case models.StatusPlanned,
		models.StatusInProgress,
		models.StatusOnApproval,
		models.StatusFinished,
		models.StatusCancelled:
	default:
		c.String(http.StatusBadRequest, "Некорректный статус проекта")
		return
	}

	// роль из сессии
	sess := sessions.Default(c)
	roleStr, _ := sess.Get("role").(string)
	role := models.UserRole(roleStr)

	if !canChangeProjectStatus(role, project.Status, status) {
		c.String(http.StatusForbidden, "Недостаточно прав для смены статуса проекта")
		return
	}

	now := time.Now()
	if status == models.StatusFinished {
		project.ActualEnd = &now
	}

	project.Status = status

	if err := database.DB.Save(&project).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка обновления статуса проекта")
		return
	}

	if uid, ok := sess.Get("user_id").(uint); ok {
		database.CreateAuditLog(uid, "project", project.ID, "status_change",
			"Статус проекта изменён на "+string(status))
	}

	c.Redirect(http.StatusFound, "/projects")
}

// Правила смены статусов (вариант B)
func canChangeProjectStatus(role models.UserRole, current, next models.ProjectStatus) bool {
	if current == next {
		return false
	}

	switch role {
	case models.RoleAdmin:
		// админ может всё
		return true

	case models.RoleSales:
		switch current {
		case models.StatusPlanned:
			// sales: запускает или отменяет
			return next == models.StatusInProgress || next == models.StatusCancelled
		case models.StatusOnApproval:
			// sales: после согласования либо назад в работу, либо отмена
			return next == models.StatusInProgress || next == models.StatusCancelled
		default:
			return false
		}

	case models.RoleEngineer:
		// инженер: только переводит "в работе" → "на согласование"
		return current == models.StatusInProgress && next == models.StatusOnApproval

	default:
		// viewer и прочие — только смотрят
		return false
	}
}

func renderProjectError(c *gin.Context, msg string) {
	var clients []models.Client
	database.DB.Order("name asc").Find(&clients)

	var assets []models.Asset
	database.DB.Order("name asc").Find(&assets)

	var engineers []models.User
	database.DB.Where("role = ?", models.RoleEngineer).Order("username asc").Find(&engineers)

	c.HTML(http.StatusBadRequest, "projects_new.html", gin.H{
		"error":     msg,
		"clients":   clients,
		"assets":    assets,
		"engineers": engineers,
	})
}

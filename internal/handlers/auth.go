package handlers

import (
	"net/http"
	"strings"

	"ib-integrator/internal/database"
	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func ShowRegister(c *gin.Context) {
	render(c, http.StatusOK, "register.html", gin.H{"error": ""})
}

type registerForm struct {
	Username string `form:"username"`
	Password string `form:"password"`
	Role     string `form:"role"`
}

func Register(c *gin.Context) {
	var form registerForm
	if err := c.ShouldBind(&form); err != nil {
		render(c, http.StatusBadRequest, "register.html", gin.H{"error": "Некорректные данные"})
		return
	}

	form.Username = strings.TrimSpace(form.Username)
	if len(form.Username) < 3 || len(form.Password) < 6 {
		render(c, http.StatusBadRequest, "register.html", gin.H{"error": "Слишком короткий логин или пароль"})
		return
	}

	role := models.UserRole(form.Role)

	// через форму можно регистрировать только sales / engineer / viewer
	switch role {
	case models.RoleSales, models.RoleEngineer, models.RoleViewer:
		// ок
	default:
		render(c, http.StatusBadRequest, "register.html", gin.H{"error": "Неверная роль"})
		return
	}

	var existing models.User
	if err := database.DB.Where("username = ?", form.Username).First(&existing).Error; err == nil {
		render(c, http.StatusBadRequest, "register.html", gin.H{"error": "Пользователь уже существует"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	user := models.User{
		Username:     form.Username,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		render(c, http.StatusInternalServerError, "register.html", gin.H{"error": "Ошибка сохранения пользователя"})
		return
	}

	// Можно при желании писать в аудит создание пользователя
	// database.CreateAuditLog(user.ID, "user", user.ID, "create", "Создан пользователь "+user.Username)

	c.Redirect(http.StatusFound, "/login")
}

func ShowLogin(c *gin.Context) {
	render(c, http.StatusOK, "login.html", gin.H{"error": ""})
}

type loginForm struct {
	Username string `form:"username"`
	Password string `form:"password"`
}

func Login(c *gin.Context) {
	var form loginForm
	if err := c.ShouldBind(&form); err != nil {
		render(c, http.StatusBadRequest, "login.html", gin.H{"error": "Некорректные данные"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", form.Username).First(&user).Error; err != nil {
		render(c, http.StatusBadRequest, "login.html", gin.H{"error": "Неверный логин или пароль"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(form.Password)); err != nil {
		render(c, http.StatusBadRequest, "login.html", gin.H{"error": "Неверный логин или пароль"})
		return
	}

	sess := sessions.Default(c)
	sess.Set("user_id", user.ID)
	sess.Set("role", string(user.Role))
	_ = sess.Save()

	c.Redirect(http.StatusFound, "/clients")
}

func Logout(c *gin.Context) {
	sess := sessions.Default(c)
	sess.Clear()
	_ = sess.Save()
	c.Redirect(http.StatusFound, "/login")
}

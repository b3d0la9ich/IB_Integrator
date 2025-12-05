package server

import (
	"html/template"
	"net/http"

	"ib-integrator/internal/config"
	"ib-integrator/internal/handlers"
	"ib-integrator/internal/middleware"
	"ib-integrator/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func maskEmail(email string) string {
	runes := []rune(email)
	atIdx := -1
	for i, r := range runes {
		if r == '@' {
			atIdx = i
			break
		}
	}
	if atIdx <= 0 {
		return "***"
	}
	prefix := string(runes[:atIdx])
	domain := string(runes[atIdx:])
	if len(prefix) <= 2 {
		return prefix + "***" + domain
	}
	return string(runes[0:2]) + "***" + domain
}

func maskPhone(phone string) string {
	runes := []rune(phone)
	n := len(runes)
	if n <= 4 {
		return "***"
	}
	masked := make([]rune, n)
	for i := range runes {
		if i >= n-2 {
			masked[i] = runes[i]
		} else {
			masked[i] = '*'
		}
	}
	return string(masked)
}

func NewRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	r.Static("/static", "./web/static")

	r.SetFuncMap(template.FuncMap{
		"eq":        func(a, b interface{}) bool { return a == b },
		"maskEmail": maskEmail,
		"maskPhone": maskPhone,
	})
	r.LoadHTMLGlob("web/templates/*.html")

	store := cookie.NewStore([]byte(cfg.SessionSecret))
	r.Use(sessions.Sessions("ib_session", store))

	// ГЛАВНАЯ
	r.GET("/", handlers.IndexPage)


	// auth
	r.GET("/register", handlers.ShowRegister)
	r.POST("/register", handlers.Register)
	r.GET("/login", handlers.ShowLogin)
	r.POST("/login", handlers.Login)
	r.GET("/logout", handlers.Logout)

	auth := r.Group("/")
	auth.Use(middleware.RequireAuth())

	// клиенты
	auth.GET("/clients", handlers.ListClients)
	auth.GET("/clients/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales),
		handlers.ShowNewClient,
	)
	auth.POST("/clients/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales),
		handlers.CreateClient,
	)
	auth.GET("/clients/:id", handlers.ShowClientDetail)

	// объекты защиты
	auth.GET("/assets", handlers.ListAssets)
	auth.GET("/assets/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales, models.RoleEngineer),
		handlers.ShowNewAsset,
	)
	auth.POST("/assets/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales, models.RoleEngineer),
		handlers.CreateAsset,
	)

	// проекты
	auth.GET("/projects", handlers.ListProjects)

	auth.GET("/projects/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales),
		handlers.ShowNewProject,
	)

	auth.POST("/projects/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales),
		handlers.CreateProject,
	)

	// смена статуса: admin + sales + engineer
	auth.POST("/projects/:id/status",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales, models.RoleEngineer),
		handlers.ChangeProjectStatus,
	)


	// аудит
	auth.GET("/audit",
		middleware.RequireRole(models.RoleAdmin, models.RoleViewer),
		handlers.ListAuditLogs,
	)

	// health
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	return r
}

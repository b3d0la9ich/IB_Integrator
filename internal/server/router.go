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

	r.Use(middleware.InjectUser())

	// ГЛАВНАЯ
	r.GET("/", handlers.IndexPage)

	// AUTH
	r.GET("/register", handlers.ShowRegister)
	r.POST("/register", handlers.Register)
	r.GET("/login", handlers.ShowLogin)
	r.POST("/login", handlers.Login)
	r.GET("/logout", handlers.Logout)

	auth := r.Group("/")
	auth.Use(middleware.RequireAuth())

	// КЛИЕНТЫ
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

	// редактирование клиентов — только админ
	auth.GET("/clients/:id/edit",
		middleware.RequireRole(models.RoleAdmin),
		handlers.ShowEditClient,
	)
	auth.POST("/clients/:id/edit",
		middleware.RequireRole(models.RoleAdmin),
		handlers.UpdateClient,
	)


	// ОБЪЕКТЫ ЗАЩИТЫ
	auth.GET("/assets", handlers.ListAssets)

	auth.GET("/assets/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales),
		handlers.ShowNewAsset,
	)
	auth.POST("/assets/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleSales),
		handlers.CreateAsset,
	)

	// редактирование объектов защиты — только админ
	auth.GET("/assets/:id/edit",
		middleware.RequireRole(models.RoleAdmin),
		handlers.ShowEditAsset,
	)
	auth.POST("/assets/:id/edit",
		middleware.RequireRole(models.RoleAdmin),
		handlers.UpdateAsset,
	)


	// ====== УГРОЗЫ И МЕРЫ ЗАЩИТЫ ======
	// каталог (admin + engineer)
	auth.GET("/threats",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.ListThreatsAndMeasures,
	)

	auth.GET("/threats/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.ShowNewThreat,
	)
	auth.POST("/threats/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.CreateThreat,
	)

	auth.GET("/measures/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.ShowNewMeasure,
	)
	auth.POST("/measures/new",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.CreateMeasure,
	)

	// угрозы конкретного объекта защиты
	auth.GET("/assets/:id/threats",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.ShowAssetThreats,
	)
	auth.POST("/assets/:id/threats/add",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.AddAssetThreat,
	)
	auth.POST("/assets/:id/threats/:link_id/delete",
		middleware.RequireRole(models.RoleAdmin, models.RoleEngineer),
		handlers.DeleteAssetThreat,
	)

	// АУДИТ
	auth.GET("/audit",
		middleware.RequireRole(models.RoleAdmin, models.RoleViewer),
		handlers.ListAuditLogs,
	)

	// HEALTHCHECK
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	return r
}

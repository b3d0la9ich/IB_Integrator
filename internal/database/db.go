package database

import (
	"log"
	"os"
	"time"

	"ib-integrator/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dsn string) {
	var err error

	const maxAttempts = 10
	for i := 1; i <= maxAttempts; i++ {
		log.Printf("trying to connect to DB (attempt %d/%d)...", i, maxAttempts)

		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("connected to DB successfully")
			break
		}

		log.Printf("failed to connect to DB: %v", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("failed to connect to db after %d attempts: %v", maxAttempts, err)
	}

	// миграции
	err = DB.AutoMigrate(
		&models.User{},
		&models.Client{},
		&models.Asset{},
		&models.Project{},
		&models.AuditLog{},
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	// создаём дефолтного админа и пару тестовых пользователей
	createDefaultAdmin()
	seedDefaultUsers()
}

// админ только из кода/конфига
func createDefaultAdmin() {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin@ib.local"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "Admin123!"
	}

	var count int64
	if err := DB.Model(&models.User{}).
		Where("role = ?", models.RoleAdmin).
		Count(&count).Error; err != nil {
		log.Printf("failed to check admin user: %v", err)
		return
	}
	if count > 0 {
		// админ уже есть — ничего не делаем
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash default admin password: %v", err)
		return
	}

	admin := models.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         models.RoleAdmin,
	}

	if err := DB.Create(&admin).Error; err != nil {
		log.Printf("failed to create default admin: %v", err)
		return
	}

	log.Printf("created default admin user: %s (password: %s)", username, password)
}

// пара тестовых аккаунтов для демо (sales и engineer)
func seedDefaultUsers() {
	type seedUser struct {
		Username string
		Password string
		Role     models.UserRole
	}

	users := []seedUser{
		{
			Username: "sales@ib.local",
			Password: "Sales123!",
			Role:     models.RoleSales,
		},
		{
			Username: "eng@ib.local",
			Password: "Eng123!",
			Role:     models.RoleEngineer,
		},
	}

	for _, u := range users {
		var count int64
		if err := DB.Model(&models.User{}).
			Where("username = ?", u.Username).
			Count(&count).Error; err != nil {
			log.Printf("failed to check seed user %s: %v", u.Username, err)
			continue
		}
		if count > 0 {
			// уже есть — пропускаем
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("failed to hash password for %s: %v", u.Username, err)
			continue
		}

		user := models.User{
			Username:     u.Username,
			PasswordHash: string(hash),
			Role:         u.Role,
		}

		if err := DB.Create(&user).Error; err != nil {
			log.Printf("failed to create seed user %s: %v", u.Username, err)
			continue
		}

		log.Printf("created seed user: %s (role=%s, password=%s)", u.Username, u.Role, u.Password)
	}
}

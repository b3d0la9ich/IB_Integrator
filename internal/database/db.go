package database

import (
	"errors"
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
		&models.AuditLog{},

		// 💾 новые таблицы каталога угроз и мер
		&models.Threat{},
		&models.ControlMeasure{},
		&models.AssetThreat{},
		&models.ThreatMeasure{}, // <--- СВЯЗЬ УГРОЗА → МЕРА
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	// 📌 сидинг каталога угроз и мер защиты + связок "угроза → мера"
	if err := seedThreatsAndMeasures(); err != nil {
		log.Fatalf("failed to seed threats/measures: %v", err)
	}

	// создаём дефолтного админа и пару тестовых пользователей
	createDefaultAdmin()
	seedDefaultUsers()
}

// seedThreatsAndMeasures заполняет базовый каталог угроз и мер защиты.
// Вызывается один раз при старте (после AutoMigrate).
func seedThreatsAndMeasures() error {
	// --- Базовый каталог угроз (пример: STRIDE + общие ИБ-угрозы для БД/систем управления ИБ) ---
	baseThreats := []models.Threat{
		{
			Code:        "STRIDE-S",
			Name:        "Подмена личности (Spoofing)",
			Category:    "STRIDE",
			Description: "Угроза подмены субъекта доступа (аккаунты пользователей, сервисов, админских учёток).",
		},
		{
			Code:        "STRIDE-T",
			Name:        "Подмена/искажение данных (Tampering)",
			Category:    "STRIDE",
			Description: "Нарушение целостности данных в БД, логах, конфигурациях систем.",
		},
		{
			Code:        "DB-LEAK",
			Name:        "Несанкционированное раскрытие данных БД",
			Category:    "Конфиденциальность",
			Description: "Утечка данных клиентов и объектов защиты через компрометацию учётных данных или уязвимости приложений.",
		},
		{
			Code:        "DB-DOS",
			Name:        "Нарушение доступности БД",
			Category:    "Доступность",
			Description: "Вывод из строя сервиса интегратора или СУБД, отказ в обслуживании.",
		},
		{
			Code:        "ADM-MISCONF",
			Name:        "Ошибочное администрирование и нехватка контроля",
			Category:    "Организационные",
			Description: "Неверные настройки прав, отсутствие аудита действий администраторов и инженеров.",
		},
	}

	for _, t := range baseThreats {
		var existing models.Threat
		err := DB.Where("code = ?", t.Code).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := DB.Create(&t).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// --- Базовый каталог мер защиты ---
	baseMeasures := []models.ControlMeasure{
		{
			Code:     "FW-NET-SEGMENT",
			Name:     "Сетевой экран и сегментация",
			Standard: "ФСТЭК, ГОСТ; best practices ИБ",
			Description: "Разделение сетей, фильтрация трафика между сегментами, " +
				"ограничение доступа к БД и служебным сервисам.",
		},
		{
			Code:     "AUTH-RBAC",
			Name:     "Ролевое управление доступом",
			Standard: "ФСТЭК, ISO 27001 A.9",
			Description: "Роли admin/sales/engineer/viewer, ограничение административных операций и доступа к данным.",
		},
		{
			Code:     "LOG-AUDIT",
			Name:     "Журналирование и аудит действий",
			Standard: "ФСТЭК, ГОСТ, внутренние политики ИБ",
			Description: "Регистрация операций с клиентами, объектами защиты, проектами; анализ подозрительной активности.",
		},
		{
			Code:     "DB-BACKUP",
			Name:     "Резервное копирование БД",
			Standard: "ГОСТ по резервированию, best practices",
			Description: "Регулярные резервные копии, проверка восстановления, хранение резервов в защищённой зоне.",
		},
		{
			Code:     "SEC-CODE-REV",
			Name:     "Контроль безопасности приложений",
			Standard: "OWASP, внутренние стандарты",
			Description: "Анализ кода, устранение SQL-инъекций и XSS, безопасная конфигурация ORM и инфраструктуры.",
		},
	}

	for _, m := range baseMeasures {
		var existing models.ControlMeasure
		err := DB.Where("code = ?", m.Code).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := DB.Create(&m).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// --- Связки "угроза → рекомендуемые меры защиты" ---
	// связываем по code, чтобы не завязываться на ID
	type link struct {
		ThreatCode  string
		MeasureCode string
	}

	links := []link{
		// Подмена личности → RBAC + аудит
		{"STRIDE-S", "AUTH-RBAC"},
		{"STRIDE-S", "LOG-AUDIT"},

		// Тамперинг данных → контроль кода + аудит
		{"STRIDE-T", "SEC-CODE-REV"},
		{"STRIDE-T", "LOG-AUDIT"},

		// Утечка БД → RBAC + сегментация + аудит
		{"DB-LEAK", "AUTH-RBAC"},
		{"DB-LEAK", "FW-NET-SEGMENT"},
		{"DB-LEAK", "LOG-AUDIT"},

		// Доступность БД → бэкапы + сегментация
		{"DB-DOS", "DB-BACKUP"},
		{"DB-DOS", "FW-NET-SEGMENT"},

		// Ошибочное администрирование → аудит + RBAC
		{"ADM-MISCONF", "LOG-AUDIT"},
		{"ADM-MISCONF", "AUTH-RBAC"},
	}

	for _, l := range links {
		var th models.Threat
		if err := DB.Where("code = ?", l.ThreatCode).First(&th).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		var m models.ControlMeasure
		if err := DB.Where("code = ?", l.MeasureCode).First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		var cnt int64
		if err := DB.Model(&models.ThreatMeasure{}).
			Where("threat_id = ? AND measure_id = ?", th.ID, m.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			continue
		}

		tm := models.ThreatMeasure{
			ThreatID:  th.ID,
			MeasureID: m.ID,
		}
		if err := DB.Create(&tm).Error; err != nil {
			return err
		}
	}

	return nil
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

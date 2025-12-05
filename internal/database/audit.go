package database

import "ib-integrator/internal/models"

// helper для записи в журнал аудита
func CreateAuditLog(userID uint, entity string, entityID uint, action, details string) {
	if DB == nil {
		return
	}
	record := models.AuditLog{
		UserID:   userID,
		Entity:   entity,
		EntityID: entityID,
		Action:   action,
		Details:  details,
	}
	_ = DB.Create(&record).Error
}

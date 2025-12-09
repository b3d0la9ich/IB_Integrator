// internal/models/threat_measure.go
package models

import "gorm.io/gorm"

// ThreatMeasure — связь "угроза → рекомендуемая мера защиты"
type ThreatMeasure struct {
	gorm.Model

	ThreatID  uint
	MeasureID uint

	Threat  Threat
	Measure ControlMeasure
}

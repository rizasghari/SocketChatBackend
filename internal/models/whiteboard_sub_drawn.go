package models

import "gorm.io/gorm"

type SubDrawn struct {
	gorm.Model
	DrawnID uint   `json:"drawn_id"`
	Points  Points `json:"points" gorm:"type:jsonb"`
	Paint   Paint  `json:"paint" gorm:"type:jsonb"`
}
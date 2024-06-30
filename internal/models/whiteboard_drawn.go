package models

import (
	"gorm.io/gorm"
)

type Drawn struct {
	gorm.Model
	Drawer       uint         `json:"drawer_user_id" gorm:"column:user_id"`
	SubDrawns    *[]SubDrawn `json:"sub_drawns"`
	WhiteboardID uint         `json:"whiteboard_id"`
}

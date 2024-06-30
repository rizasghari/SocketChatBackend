package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Paint is not db entity
// Embeded in sub drawn db entity as jsonb column
type Paint struct {
	Color         int64       `json:"color"`
	StrokeWidth   float64     `json:"stroke_width"`
	StrokeCap     string      `json:"stroke_cap" `
	StrokeJoin    string      `json:"stroke_join"`
	PaintingStyle string      `json:"painting_style"`
	FilterQuality string      `json:"filter_quality"`
	BlendMode     string      `json:"blend_mode"`
	IsAntiAlias   bool        `json:"is_anti_alias"`
}

func (p *Paint) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, p)
}

func (p Paint) Value() (driver.Value, error) {
	return json.Marshal(p)
}

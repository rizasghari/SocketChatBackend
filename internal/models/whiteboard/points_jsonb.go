package whiteboard

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// To satisfay postgres jsonb data type
type Points []Point

func (p *Points) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, p)
}

func (p Points) Value() (driver.Value, error) {
	return json.Marshal(p)
}

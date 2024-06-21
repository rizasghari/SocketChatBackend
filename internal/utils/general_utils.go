package utils

import (
	"fmt"
	"strings"
	"time"
)

func StrToTime(value string) (*time.Time, error) {
	layout := "2006-01-02 15:04"

	slice := strings.Split(value, "T")
	value = fmt.Sprintf("%v %v", slice[0], slice[1][:5])
	result, err := time.Parse(layout, value)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

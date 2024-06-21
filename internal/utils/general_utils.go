package utils

import "time"

func StrToTime(value string) (*time.Time, error) {
	layout := "2006-01-02 15:04:05"
	result, err := time.Parse(layout, value)
	if err != nil {
		return nil, err
	}
	return &result, nil
}	
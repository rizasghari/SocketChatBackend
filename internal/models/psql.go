package models

type PSQL struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSL      string
	Timezone string
}

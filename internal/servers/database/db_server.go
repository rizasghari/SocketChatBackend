package database

import (
	"fmt"
	"log"
	"socketChat/configs"
	"socketChat/internal/models"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func GetDB(config *configs.Config) *gorm.DB {
	once.Do(func() {
		initialize(config)
	})
	return db
}

func initialize(config *configs.Config) {
	psql := getPSQL(config)
	dsn := fmt.Sprintf(
		"host=%v user=%v password=%v dbname=%v port=%v sslmode=%v TimeZone=%v",
		psql.Host, psql.User, psql.Password, psql.Name, psql.Port, psql.SSL, psql.Timezone,
	)
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrate()
}

func getPSQL(config *configs.Config) *models.PSQL {
	return &models.PSQL{
		Host:     config.Viper.GetString("database.host"),
		Port:     config.Viper.GetInt("database.port"),
		User:     config.Viper.GetString("database.user"),
		Password: config.Viper.GetString("database.password"),
		Name:     config.Viper.GetString("database.name"),
		SSL:      config.Viper.GetString("database.ssl"),
		Timezone: config.Viper.GetString("database.timezone"),
	}
}

func migrate() {
	err := db.AutoMigrate(
		&models.User{},
		&models.Conversation{},
		&models.ConversationMember{},
		&models.Message{},
		&models.Whiteboard{},
		&models.Drawn{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
		return
	} else {
		log.Println("Database migrated successfully")
	}
}

package singal

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type AuthUser struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Username  string    `gorm:"size:100" json:"username"`
	IsActive  bool      `gorm:"default:false" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var gAuthDB *gorm.DB

func InitAuthDB() error {
	host := gConfig.Database.Host
	port := gConfig.Database.Port
	user := gConfig.Database.Username
	password := gConfig.Database.Password
	dbname := gConfig.Database.Name

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)

	gAuthDB, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if gAuthDB == nil {
		return fmt.Errorf("failed to connect to database")
	}

	sqlDB, err := gAuthDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	err = gAuthDB.AutoMigrate(&AuthUser{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func GetAuthDB() *gorm.DB {
	return gAuthDB
}

package db

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type SensorRow struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	Value      float64   `gorm:"not null"`
	SensorType string    `gorm:"size:64;not null;index:idx_type_ts"`
	ID1        string    `gorm:"type:char(1);not null;index:idx_id1_id2_ts"`
	ID2        int       `gorm:"not null;index:idx_id1_id2_ts"`
	TS         time.Time `gorm:"type:timestamp(6);not null;index:idx_id1_id2_ts;index:idx_type_ts"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func ConnectAndMigrate() *gorm.DB {
	// Keep it simple: change DSN if needed.
	// Create DB:  CREATE DATABASE sensors CHARACTER SET utf8mb4;
	dsn := "root:password@tcp(127.0.0.1:3306)/sensors?parseTime=true&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	if err := db.AutoMigrate(&SensorRow{}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	return db
}

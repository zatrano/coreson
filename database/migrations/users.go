package migrations

import (
	"errors"
	"zatrano/models"
	"zatrano/pkg/logs"

	"gorm.io/gorm"
)

func MigrateUsersTable(db *gorm.DB) error {
	logs.SLog.Info("User tablosu için enum tipi kontrol ediliyor...")

	dropEnumQuery := `DROP TYPE IF EXISTS user_type;`
	rawDB, err := db.DB()
	if err != nil {
		return errors.New("DB instance alınamadı: " + err.Error())
	}

	_, err = rawDB.Exec(dropEnumQuery)
	if err != nil {
		return errors.New("user_type enum silinemedi: " + err.Error())
	}
	logs.SLog.Info("user_type enum başarıyla silindi.")

	createEnum := `CREATE TYPE user_type AS ENUM ('dashboard', 'panel');`
	_, err = rawDB.Exec(createEnum)
	if err != nil {
		return errors.New("user_type enum oluşturulamadı: " + err.Error())
	}
	logs.SLog.Info("user_type enum başarıyla oluşturuldu.")

	logs.SLog.Info("User tablosu migrate ediliyor...")
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return errors.New("User tablosu migrate edilemedi: " + err.Error())
	}

	logs.SLog.Info("User tablosu migrate işlemi tamamlandı.")
	return nil
}

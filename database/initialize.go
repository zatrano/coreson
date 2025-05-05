package database

import (
	"zatrano/database/migrations"
	"zatrano/database/seeders"
	"zatrano/models"
	"zatrano/pkg/logs"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Initialize(db *gorm.DB, migrate bool, seed bool) {
	if !migrate && !seed {
		logs.SLog.Info("Migrate veya seed bayrağı belirtilmedi, işlem yapılmayacak.")
		return
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logs.Log.Fatal("Veritabanı başlatma işlemi başarısız oldu, geri alındı (panic)", zap.Any("panic_info", r))
		}
		if tx.Error != nil && tx.Error != gorm.ErrInvalidTransaction {
			logs.SLog.Warn("Başlatma sırasında hata oluştuğu için işlem geri alınıyor.")
			tx.Rollback()
		}
	}()

	logs.SLog.Info("Veritabanı başlatma işlemi başlıyor...")

	if migrate {
		logs.SLog.Info("Migrasyonlar çalıştırılıyor...")
		if err := RunMigrationsInOrder(tx); err != nil {
			tx.Rollback()
			logs.Log.Fatal("Migrasyon başarısız oldu", zap.Error(err))
		}
		logs.SLog.Info("Migrasyonlar tamamlandı.")
	} else {
		logs.SLog.Info("Migrate bayrağı belirtilmedi, migrasyon adımı atlanıyor.")
	}

	if seed {
		logs.SLog.Info("Seeder'lar çalıştırılıyor...")
		if err := CheckAndRunSeeders(tx); err != nil {
			tx.Rollback()
			logs.Log.Fatal("Seeding başarısız oldu", zap.Error(err))
		}
		logs.SLog.Info("Seeder'lar tamamlandı.")
	} else {
		logs.SLog.Info("Seed bayrağı belirtilmedi, seeder adımı atlanıyor.")
	}

	logs.SLog.Info("İşlem commit ediliyor...")
	if err := tx.Commit().Error; err != nil {
		logs.Log.Fatal("Commit başarısız oldu", zap.Error(err))
	}

	logs.SLog.Info("Veritabanı başlatma işlemi başarıyla tamamlandı")
}

func RunMigrationsInOrder(db *gorm.DB) error {
	logs.SLog.Info(" -> User migrasyonları çalıştırılıyor...")
	if err := migrations.MigrateUsersTable(db); err != nil {
		logs.Log.Error("Users tablosu migrasyonu başarısız oldu", zap.Error(err))
		return err
	}
	logs.SLog.Info(" -> User migrasyonları tamamlandı.")

	logs.SLog.Info("Tüm migrasyonlar başarıyla çalıştırıldı.")
	return nil
}

func CheckAndRunSeeders(db *gorm.DB) error {
	systemUser := seeders.GetSystemUserConfig()
	var existingUser models.User
	result := db.Where("account = ? AND type = ?", systemUser.Account, models.Dashboard).First(&existingUser)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			logs.SLog.Info("Sistem kullanıcısı oluşturuluyor: %s (%s)...", systemUser.Name, systemUser.Account)
			if err := seeders.SeedSystemUser(db); err != nil {
				logs.Log.Error("Sistem kullanıcısı seed edilemedi", zap.Error(err))
				return err
			}
			logs.SLog.Info(" -> Sistem kullanıcısı oluşturuldu.")
		} else {
			logs.Log.Error("Sistem kullanıcısı kontrol edilirken hata", zap.Error(result.Error))
			return result.Error
		}
	} else {
		logs.SLog.Info("Sistem kullanıcısı '%s' (%s) zaten mevcut, oluşturma adımı atlanıyor.",
			existingUser.Name, existingUser.Account)
		logs.SLog.Info("Mevcut sistem kullanıcısı '%s' için güncelleme kontrolü yapılıyor...", existingUser.Account)
		if err := seeders.SeedSystemUser(db); err != nil {
			logs.Log.Error("Mevcut sistem kullanıcısı güncellenirken/kontrol edilirken hata", zap.Error(err))
			return err
		}

	}
	return nil
}

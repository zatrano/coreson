package main

import (
	"flag"

	"zatrano/configs"
	"zatrano/database"
	"zatrano/pkg/logs"
)

func main() {
	logs.InitLogger()
	defer logs.SyncLogger()
	migrateFlag := flag.Bool("migrate", false, "Veritabanı başlatma işlemini çalıştır (migrasyonları içerir)")
	seedFlag := flag.Bool("seed", false, "Veritabanı başlatma işlemini çalıştır (seederları içerir)")
	flag.Parse()

	configs.InitDB()
	defer configs.CloseDB()

	db := configs.GetDB()

	logs.SLog.Info("Veritabanı başlatma işlemi çalıştırılıyor...")
	database.Initialize(db, *migrateFlag, *seedFlag)

	logs.SLog.Info("Veritabanı başlatma işlemi tamamlandı.")
}

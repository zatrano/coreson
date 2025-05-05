package configs

import (
	"encoding/gob"
	"time"

	"zatrano/models"
	"zatrano/pkg/env"
	"zatrano/pkg/logs"
	"zatrano/pkg/sessions"

	"github.com/gofiber/fiber/v2/middleware/session"
)

var Session *session.Store

func InitSession() {
	Session = createSessionStore()
	sessions.InitializeSessionStore(Session)
	logs.SLog.Info("Oturum (session) sistemi başlatıldı ve utils içinde kayıt edildi.")
}

func SetupSession() *session.Store {
	if Session == nil {
		logs.SLog.Warn("Session store isteniyor ancak henüz başlatılmamış, şimdi başlatılıyor.")
		InitSession()
	}
	return Session
}

func createSessionStore() *session.Store {
	sessionExpirationHours := env.GetEnvAsInt("SESSION_EXPIRATION_HOURS", 24)

	cookieSecure := env.IsProduction()

	store := session.New(session.Config{
		CookieHTTPOnly: false,
		CookieSecure:   cookieSecure,
		Expiration:     time.Duration(sessionExpirationHours) * time.Hour,
		KeyLookup:      "cookie:session_id",
		CookieSameSite: "Lax",
	})

	logs.SLog.Info("Cookie tabanlı session sistemi %d saatlik süreyle yapılandırıldı.", sessionExpirationHours)

	registerGobTypes()

	return store
}

func registerGobTypes() {
	gob.Register(models.UserType(""))
	gob.Register(&models.User{})
	logs.SLog.Debug("Session için gob türleri kaydedildi: models.UserType, *models.User")
}

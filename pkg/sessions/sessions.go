package sessions

import (
	"zatrano/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

var store *session.Store

func InitializeSessionStore(s *session.Store) {
	store = s
}

func SessionStart(c *fiber.Ctx) (*session.Session, error) {
	if store == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "session store not initialized")
	}
	return store.Get(c)
}

func GetUserTypeFromSession(sess *session.Session) (models.UserType, error) {
	userType, ok := sess.Get("user_type").(models.UserType)
	if !ok {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Geçersiz oturum veya kullanıcı tipi")
	}
	return userType, nil
}

func GetUserIDFromSession(sess *session.Session) (uint, error) {
	userID, ok := sess.Get("user_id").(uint)
	if !ok {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Geçersiz oturum veya kullanıcı ID'si")
	}
	return userID, nil
}

func GetUserStatusFromSession(sess *session.Session) (bool, error) {
	userStatus, ok := sess.Get("user_status").(bool)
	if !ok {
		return false, fiber.NewError(fiber.StatusUnauthorized, "Geçersiz oturum veya kullanıcı durumu")
	}
	return userStatus, nil

}

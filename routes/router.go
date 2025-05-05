package routes

import (
	"zatrano/configs"
	"zatrano/models"
	"zatrano/pkg/sessions"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	app.Use(logger.New())

	sessionStore := configs.SetupSession()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session", sessionStore)
		return c.Next()
	})

	registerAuthRoutes(app)
	registerDashboardRoutes(app)
	registerPanelRoutes(app)

	app.Use(rootRedirector)
}

func rootRedirector(c *fiber.Ctx) error {
	sess, err := sessions.SessionStart(c)
	if err != nil {
		return c.Redirect("/auth/login")
	}

	_, err = sessions.GetUserIDFromSession(sess)
	if err != nil {
		return c.Redirect("/auth/login")
	}

	userType, err := sessions.GetUserTypeFromSession(sess)
	if err != nil {
		return c.Redirect("/auth/login")
	}

	switch userType {
	case models.Panel:
		return c.Redirect("/panel/home")
	case models.Dashboard:
		return c.Redirect("/dashboard/home")
	default:
		return c.SendString("Geçersiz kullanıcı tipi")
	}
}

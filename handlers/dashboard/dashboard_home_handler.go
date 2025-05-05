package handlers

import (
	"net/http"
	"zatrano/pkg/logs"
	"zatrano/pkg/renderer"
	"zatrano/services"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type DashboardHomeHandler struct {
	userService services.IUserService
}

func NewDashboardHomeHandler() *DashboardHomeHandler {
	return &DashboardHomeHandler{
		userService: services.NewUserService(),
	}
}

func (h *DashboardHomeHandler) HomePage(c *fiber.Ctx) error {
	userCount, userErr := h.userService.GetUserCount()
	if userErr != nil {
		logs.Log.Error("Anasayfa: Kullanıcı sayısı alınamadı", zap.Error(userErr))
		userCount = 0
	}

	mapData := fiber.Map{
		"Title":     "Dashboard",
		"UserCount": userCount,
	}
	return renderer.Render(c, "dashboard/home/home", "layouts/dashboard", mapData, http.StatusOK)
}

package handlers

import (
	"errors"
	"net/http"
	"zatrano/models"
	"zatrano/pkg/flashmessages"
	"zatrano/pkg/logs"
	"zatrano/pkg/queryparams"
	"zatrano/pkg/renderer"
	"zatrano/services"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type UserHandler struct {
	userService services.IUserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: services.NewUserService(),
	}
}

func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	var params queryparams.ListParams
	if err := c.QueryParser(&params); err != nil {
		logs.Log.Warn("Kullanıcı listesi: Query parametreleri parse edilemedi, varsayılanlar kullanılıyor.", zap.Error(err))
		params = queryparams.ListParams{
			Page: queryparams.DefaultPage, PerPage: queryparams.DefaultPerPage,
			SortBy: queryparams.DefaultSortBy, OrderBy: queryparams.DefaultOrderBy,
		}
	}

	if params.Page <= 0 {
		params.Page = queryparams.DefaultPage
	}
	if params.PerPage <= 0 {
		params.PerPage = queryparams.DefaultPerPage
	}
	if params.PerPage > queryparams.MaxPerPage {
		logs.Log.Warn("Sayfa başına istenen kayıt sayısı limiti aştı, varsayılana çekildi.",
			zap.Int("requested", params.PerPage), zap.Int("max", queryparams.MaxPerPage), zap.Int("default", queryparams.DefaultPerPage))
		params.PerPage = queryparams.DefaultPerPage
	}
	if params.SortBy == "" {
		params.SortBy = queryparams.DefaultSortBy
	}
	if params.OrderBy == "" {
		params.OrderBy = queryparams.DefaultOrderBy
	}

	paginatedResult, dbErr := h.userService.GetAllUsers(params)

	renderData := fiber.Map{
		"Title":  "Kullanıcılar",
		"Result": paginatedResult,
		"Params": params,
	}
	statusCode := http.StatusOK

	if dbErr != nil {
		dbErrMsg := "Kullanıcılar getirilirken bir hata oluştu."
		logs.Log.Error("Kullanıcı listesi DB Hatası", zap.Error(dbErr))
		renderData[renderer.FlashErrorKeyView] = dbErrMsg
		renderData["Result"] = &queryparams.PaginatedResult{
			Data: []models.User{},
			Meta: queryparams.PaginationMeta{
				CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0,
			},
		}
	}

	return renderer.Render(c, "dashboard/users/list", "layouts/dashboard", renderData, statusCode)
}

func (h *UserHandler) ShowCreateUser(c *fiber.Ctx) error {
	mapData := fiber.Map{
		"Title": "Yeni Kullanıcı Ekle",
	}
	return renderer.Render(c, "dashboard/users/create", "layouts/dashboard", mapData)
}

func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	type Request struct {
		Name     string `form:"name"`
		Account  string `form:"account"`
		Password string `form:"password"`
		Status   string `form:"status"`
		Type     string `form:"type"`
	}
	var req Request

	if err := c.BodyParser(&req); err != nil {
		logs.SLog.Warnf("Kullanıcı oluşturma isteği ayrıştırılamadı: %v", err)
		mapData := fiber.Map{
			"Title":                    "Yeni Kullanıcı Ekle",
			renderer.FlashErrorKeyView: "Geçersiz veri formatı veya eksik alanlar.",
			renderer.FormDataKey:       req,
		}
		return renderer.Render(c, "dashboard/users/create", "layouts/dashboard", mapData, http.StatusBadRequest)
	}

	if req.Name == "" || req.Account == "" || req.Password == "" || req.Type == "" {
		mapData := fiber.Map{
			"Title":                    "Yeni Kullanıcı Ekle",
			renderer.FlashErrorKeyView: "Ad, Hesap Adı, Şifre ve Kullanıcı Tipi alanları zorunludur.",
			renderer.FormDataKey:       req,
		}
		return renderer.Render(c, "dashboard/users/create", "layouts/dashboard", mapData, http.StatusBadRequest)
	}

	status := req.Status == "true"

	user := models.User{
		Name:     req.Name,
		Account:  req.Account,
		Password: req.Password,
		Status:   status,
		Type:     models.UserType(req.Type),
	}

	if user.Type != models.Dashboard && user.Type != models.Panel {
		mapData := fiber.Map{
			"Title":                    "Yeni Kullanıcı Ekle",
			renderer.FlashErrorKeyView: "Geçersiz kullanıcı tipi seçildi.",
			renderer.FormDataKey:       req,
		}
		return renderer.Render(c, "dashboard/users/create", "layouts/dashboard", mapData, http.StatusBadRequest)
	}

	if err := h.userService.CreateUser(c.UserContext(), &user); err != nil {
		logs.Log.Error("Kullanıcı oluşturulamadı (Servis Hatası)", zap.String("account", req.Account), zap.Error(err))
		errMsg := "Kullanıcı oluşturulamadı: " + err.Error()
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errors.New("parola zorunlu")) || errors.Is(err, errors.New("parola şifreleme hatası")) {
			statusCode = http.StatusBadRequest
		}

		mapData := fiber.Map{
			"Title":                    "Yeni Kullanıcı Ekle",
			renderer.FlashErrorKeyView: errMsg,
			renderer.FormDataKey:       req,
		}
		return renderer.Render(c, "dashboard/users/create", "layouts/dashboard", mapData, statusCode)
	}

	_ = flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı başarıyla oluşturuldu.")
	return c.Redirect("/dashboard/users", fiber.StatusFound)
}

func (h *UserHandler) ShowUpdateUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		logs.Log.Warn("Kullanıcı güncelleme formu: Geçersiz ID parametresi", zap.String("param", c.Params("id")))
		_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Geçersiz kullanıcı ID'si.")
		return c.Redirect("/dashboard/users", fiber.StatusSeeOther)
	}
	userID := uint(id)

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		var errMsg string
		if errors.Is(err, errors.New("kayıt bulunamadı")) {
			logs.Log.Warn("Kullanıcı güncelleme formu: Kullanıcı bulunamadı", zap.Uint("user_id", userID))
			errMsg = "Düzenlenecek kullanıcı bulunamadı."
		} else {
			logs.Log.Error("Kullanıcı güncelleme formu: Kullanıcı alınamadı (Servis Hatası)", zap.Uint("user_id", userID), zap.Error(err))
			errMsg = "Kullanıcı bilgileri alınırken hata oluştu."
		}
		_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect("/dashboard/users", fiber.StatusSeeOther)
	}

	mapData := fiber.Map{
		"Title": "Kullanıcı Düzenle",
		"User":  user,
	}

	return renderer.Render(c, "dashboard/users/update", "layouts/dashboard", mapData)
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		logs.Log.Warn("Kullanıcı güncelleme: Geçersiz ID parametresi", zap.String("param", c.Params("id")))
		_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Geçersiz kullanıcı ID'si.")
		return c.Redirect("/dashboard/users", fiber.StatusSeeOther)
	}
	userID := uint(id)
	redirectPathOnSuccess := "/dashboard/users"

	type Request struct {
		Name     string `form:"name"`
		Account  string `form:"account"`
		Password string `form:"password"`
		Status   string `form:"status"`
		Type     string `form:"type"`
	}
	var req Request

	if err := c.BodyParser(&req); err != nil {
		logs.Log.Warn("Kullanıcı güncelleme: Form verileri okunamadı", zap.Uint("user_id", userID), zap.Error(err))
		user, _ := h.userService.GetUserByID(userID)
		mapData := fiber.Map{
			"Title":                    "Kullanıcı Düzenle",
			renderer.FlashErrorKeyView: "Form verileri okunamadı veya eksik.",
			renderer.FormDataKey:       req,
			"User":                     user,
		}
		return renderer.Render(c, "dashboard/users/update", "layouts/dashboard", mapData, http.StatusBadRequest)
	}

	if req.Name == "" || req.Account == "" || req.Type == "" {
		user, _ := h.userService.GetUserByID(userID)
		mapData := fiber.Map{
			"Title":                    "Kullanıcı Düzenle",
			renderer.FlashErrorKeyView: "Ad, Hesap Adı ve Kullanıcı Tipi alanları zorunludur.",
			renderer.FormDataKey:       req,
			"User":                     user,
		}
		return renderer.Render(c, "dashboard/users/update", "layouts/dashboard", mapData, http.StatusBadRequest)
	}

	userType := models.UserType(req.Type)
	if userType != models.Dashboard && userType != models.Panel {
		user, _ := h.userService.GetUserByID(userID)
		mapData := fiber.Map{
			"Title":                    "Kullanıcı Düzenle",
			renderer.FlashErrorKeyView: "Geçersiz kullanıcı tipi seçildi.",
			renderer.FormDataKey:       req,
			"User":                     user,
		}
		return renderer.Render(c, "dashboard/users/update", "layouts/dashboard", mapData, http.StatusBadRequest)
	}

	status := req.Status == "true"

	userUpdateData := &models.User{
		Name:    req.Name,
		Account: req.Account,
		Status:  status,
		Type:    userType,
	}
	if req.Password != "" {
		userUpdateData.Password = req.Password
	}

	if err := h.userService.UpdateUser(c.UserContext(), userID, userUpdateData); err != nil {
		errMsg := "Kullanıcı güncellenemedi: " + err.Error()
		statusCode := http.StatusInternalServerError

		if errors.Is(err, errors.New("kayıt bulunamadı")) {
			logs.Log.Warn("Kullanıcı güncelleme: Kullanıcı bulunamadı (Servis hatası)", zap.Uint("user_id", userID))
			errMsg = "Güncellenecek kullanıcı bulunamadı."
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
			return c.Redirect(redirectPathOnSuccess, fiber.StatusSeeOther)
		} else if errors.Is(err, errors.New("parola güncelleme hatası")) || errors.Is(err, errors.New("parola şifreleme hatası")) {
			statusCode = http.StatusBadRequest
		}

		logs.Log.Error("Kullanıcı güncelleme: Handler'da servis hatası yakalandı", zap.Uint("user_id", userID), zap.Error(err))
		user, _ := h.userService.GetUserByID(userID)
		mapData := fiber.Map{
			"Title":                    "Kullanıcı Düzenle",
			renderer.FlashErrorKeyView: errMsg,
			renderer.FormDataKey:       req,
			"User":                     user,
		}
		return renderer.Render(c, "dashboard/users/update", "layouts/dashboard", mapData, statusCode)
	}

	_ = flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı başarıyla güncellendi.")
	return c.Redirect(redirectPathOnSuccess, fiber.StatusFound)
}

func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		logs.Log.Warn("Kullanıcı silme: Geçersiz ID parametresi", zap.String("param", c.Params("id")))
		_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Geçersiz kullanıcı ID'si.")
		return c.Redirect("/dashboard/users", fiber.StatusSeeOther)
	}
	userID := uint(id)

	if err := h.userService.DeleteUser(c.UserContext(), userID); err != nil {
		var errMsg string
		if errors.Is(err, errors.New("kayıt bulunamadı")) {
			logs.Log.Warn("Kullanıcı silme: Kullanıcı bulunamadı", zap.Uint("user_id", userID))
			errMsg = "Silinecek kullanıcı bulunamadı."
		} else {
			logs.Log.Error("Kullanıcı silme: Servis hatası", zap.Uint("user_id", userID), zap.Error(err))
			errMsg = "Kullanıcı silinemedi: " + err.Error()
		}
		_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect("/dashboard/users", fiber.StatusSeeOther)
	}

	_ = flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı başarıyla silindi.")
	return c.Redirect("/dashboard/users", fiber.StatusFound)
}

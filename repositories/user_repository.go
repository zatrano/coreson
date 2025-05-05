package repositories

import (
	"context"
	"errors"
	"strings"

	"zatrano/configs"
	"zatrano/models"
	"zatrano/pkg/logs"
	"zatrano/pkg/queryparams"
	"zatrano/pkg/turkishsearch"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IUserRepository interface {
	GetAll(params queryparams.ListParams) ([]models.User, int64, error)
	GetByID(id uint) (*models.User, error)
	GetCount() (int64, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, id uint, data map[string]interface{}, updatedByID uint) error
	Delete(ctx context.Context, id uint) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() IUserRepository {
	return &UserRepository{db: configs.GetDB()}
}

func (r *UserRepository) GetAll(params queryparams.ListParams) ([]models.User, int64, error) {
	var users []models.User
	var totalCount int64

	query := r.db.Model(&models.User{})

	if params.Name != "" {
		sqlQueryFragment, queryParams := turkishsearch.SQLFilter("name", params.Name)
		query = query.Where(sqlQueryFragment, queryParams...)
	}

	err := query.Count(&totalCount).Error
	if err != nil {
		logs.Log.Error("Kullanıcı sayısı alınırken hata (GelAll)", zap.Error(err))
		return nil, 0, err
	}

	if totalCount == 0 {
		return users, 0, nil
	}

	sortBy := params.SortBy
	orderBy := strings.ToLower(params.OrderBy)
	if orderBy != "asc" && orderBy != "desc" {
		orderBy = queryparams.DefaultOrderBy
	}
	allowedSortColumns := map[string]bool{"id": true, "name": true, "account": true, "created_at": true, "status": true, "type": true}
	if _, ok := allowedSortColumns[sortBy]; !ok {
		sortBy = queryparams.DefaultSortBy
	}
	orderClause := sortBy + " " + orderBy
	query = query.Order(orderClause)

	query = query.Preload(clause.Associations)

	offset := params.CalculateOffset()
	query = query.Limit(params.PerPage).Offset(offset)

	err = query.Find(&users).Error
	if err != nil {
		logs.Log.Error("Kullanıcı verisi çekilirken hata (GelAll)", zap.Error(err))
		return nil, totalCount, err
	}

	return users, totalCount, nil
}

func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.Preload(clause.Associations).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("kayıt bulunamadı")
		}
		logs.Log.Error("GetByID sırasında DB hatası", zap.Uint("user_id", id), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	if err != nil {
		logs.Log.Error("Count sırasında DB hatası", zap.Error(err))
	}
	return count, err
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		logs.Log.Error("Create sırasında DB hatası", zap.Any("user_account", user.Account), zap.Error(result.Error))
	}
	return result.Error
}

func (r *UserRepository) Update(ctx context.Context, id uint, data map[string]interface{}, updatedByID uint) error {
	if len(data) == 0 {
		logs.Log.Debug("UserRepository.Update: Güncellenecek veri yok.", zap.Uint("user_id", id))
		return nil
	}

	if updatedByID != 0 {
		data["updated_by"] = updatedByID
	} else {
		logs.Log.Warn("UserRepository.Update: Geçersiz updatedByID (0) alındı, updated_by alanı ayarlanamadı.",
			zap.Uint("target_user_id", id))
	}

	result := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(data)

	if result.Error != nil {
		logs.Log.Error("Update sırasında DB hatası", zap.Uint("user_id", id), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		logs.Log.Warn("UserRepository.Update: Kayıt bulunamadı veya hiçbir alan değişmedi.",
			zap.Uint("user_id", id),
			zap.Int64("rows_affected", result.RowsAffected))
		return errors.New("kayıt bulunamadı")
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	var user models.User

	findTx := r.db.WithContext(ctx).First(&user, id)
	if findTx.Error != nil {
		if errors.Is(findTx.Error, gorm.ErrRecordNotFound) {
			logs.Log.Warn("UserRepository.Delete: Silinecek kullanıcı bulunamadı", zap.Uint("user_id", id))
			return errors.New("kayıt bulunamadı")
		}
		logs.Log.Error("Delete sırasında kullanıcı bulunurken DB hatası", zap.Uint("user_id", id), zap.Error(findTx.Error))
		return findTx.Error
	}

	userID, ok := ctx.Value("user_id").(uint)
	if !ok || userID == 0 {
		return errors.New("Delete: Context içinde user_id yok veya geçersiz")
	}

	updateTx := r.db.WithContext(ctx).Model(&user).Update("deleted_by", userID)
	if updateTx.Error != nil {
		logs.Log.Error("deleted_by güncellenirken hata", zap.Error(updateTx.Error))
		return updateTx.Error
	}

	deleteTx := r.db.WithContext(ctx).Delete(&user)
	if deleteTx.Error != nil {
		logs.Log.Error("Delete sırasında DB hatası", zap.Uint("user_id", user.ID), zap.Error(deleteTx.Error))
		return deleteTx.Error
	}

	if deleteTx.RowsAffected == 0 {
		logs.Log.Warn("UserRepository.Delete: Silme işlemi 0 satırı etkiledi", zap.Uint("user_id", user.ID))
	}

	return nil
}

var _ IUserRepository = (*UserRepository)(nil)

package repository

import (
	"errors"

	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// UserRepository 用户数据访问接口
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint64) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint64) error
	List(offset, limit int) ([]*models.User, int64, error)
	GetAll() ([]*models.User, error)
}

// userRepository 用户数据访问实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(id uint64) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete 删除用户（使用事务，清理关联数据）
func (r *userRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 获取用户的所有账号ID
		var accountIDs []uint64
		if err := tx.Model(&models.TGAccount{}).Where("user_id = ?", id).Pluck("id", &accountIDs).Error; err != nil {
			return err
		}

		// 2. 删除账号关联的任务日志
		if len(accountIDs) > 0 {
			if err := tx.Where("account_id IN ?", accountIDs).Delete(&models.TaskLog{}).Error; err != nil {
				return err
			}
		}

		// 3. 删除用户的任务
		var taskIDs []uint64
		if err := tx.Model(&models.Task{}).Where("user_id = ?", id).Pluck("id", &taskIDs).Error; err != nil {
			return err
		}
		if len(taskIDs) > 0 {
			if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.TaskLog{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id = ?", id).Delete(&models.Task{}).Error; err != nil {
				return err
			}
		}

		// 4. 删除用户的账号
		if err := tx.Where("user_id = ?", id).Delete(&models.TGAccount{}).Error; err != nil {
			return err
		}

		// 5. 删除用户的代理
		if err := tx.Where("user_id = ?", id).Delete(&models.ProxyIP{}).Error; err != nil {
			return err
		}

		// 6. 删除用户的批量任务
		if err := tx.Where("user_id = ?", id).Delete(&models.BatchJob{}).Error; err != nil {
			return err
		}

		// 7. 删除用户的验证码会话
		if err := tx.Where("user_id = ?", id).Delete(&models.VerifyCodeSession{}).Error; err != nil {
			return err
		}

		// 8. 最后删除用户
		return tx.Delete(&models.User{}, id).Error
	})
}

// List 获取用户列表
func (r *userRepository) List(offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// 获取总数
	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := r.db.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetAll 获取所有用户
func (r *userRepository) GetAll() ([]*models.User, error) {
	var users []*models.User
	err := r.db.Find(&users).Error
	return users, err
}

package repository

import (
	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// BatchRepository 批量操作仓库接口
type BatchRepository interface {
	Create(job *models.BatchJob) error
	GetByID(id uint64) (*models.BatchJob, error)
	GetByUserIDAndID(userID, jobID uint64) (*models.BatchJob, error)
	GetByUserID(userID uint64, offset, limit int) ([]*models.BatchJob, int64, error)
	Update(job *models.BatchJob) error
	Delete(jobID uint64) error
	GetRunningJobs() ([]*models.BatchJob, error)
	GetJobsByStatus(status string) ([]*models.BatchJob, error)
}

// batchRepository GORM实现
type batchRepository struct {
	db *gorm.DB
}

// NewBatchRepository 创建批量操作仓库
func NewBatchRepository(db *gorm.DB) BatchRepository {
	return &batchRepository{db: db}
}

// Create 创建批量任务
func (r *batchRepository) Create(job *models.BatchJob) error {
	return r.db.Create(job).Error
}

// GetByID 根据ID获取批量任务
func (r *batchRepository) GetByID(id uint64) (*models.BatchJob, error) {
	var job models.BatchJob
	err := r.db.Where("id = ?", id).First(&job).Error
	return &job, err
}

// GetByUserIDAndID 根据用户ID和任务ID获取批量任务
func (r *batchRepository) GetByUserIDAndID(userID, jobID uint64) (*models.BatchJob, error) {
	var job models.BatchJob
	err := r.db.Where("user_id = ? AND id = ?", userID, jobID).First(&job).Error
	return &job, err
}

// GetByUserID 根据用户ID获取批量任务列表
func (r *batchRepository) GetByUserID(userID uint64, offset, limit int) ([]*models.BatchJob, int64, error) {
	var jobs []*models.BatchJob
	var total int64

	query := r.db.Where("user_id = ?", userID)

	// 获取总数
	if err := query.Model(&models.BatchJob{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&jobs).Error

	return jobs, total, err
}

// Update 更新批量任务
func (r *batchRepository) Update(job *models.BatchJob) error {
	return r.db.Save(job).Error
}

// Delete 删除批量任务
func (r *batchRepository) Delete(jobID uint64) error {
	return r.db.Delete(&models.BatchJob{}, jobID).Error
}

// GetRunningJobs 获取运行中的任务
func (r *batchRepository) GetRunningJobs() ([]*models.BatchJob, error) {
	var jobs []*models.BatchJob
	err := r.db.Where("status = ?", "running").Find(&jobs).Error
	return jobs, err
}

// GetJobsByStatus 根据状态获取任务
func (r *batchRepository) GetJobsByStatus(status string) ([]*models.BatchJob, error) {
	var jobs []*models.BatchJob
	err := r.db.Where("status = ?", status).Find(&jobs).Error
	return jobs, err
}

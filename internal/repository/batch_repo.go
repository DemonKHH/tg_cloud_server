package repository

import (
	"gorm.io/gorm"

	"tg_cloud_server/internal/services"
)

// BatchRepository 批量操作仓库接口
type BatchRepository interface {
	Create(job *services.BatchJob) error
	GetByID(id uint64) (*services.BatchJob, error)
	GetByUserIDAndID(userID, jobID uint64) (*services.BatchJob, error)
	GetByUserID(userID uint64, offset, limit int) ([]*services.BatchJob, int64, error)
	Update(job *services.BatchJob) error
	Delete(jobID uint64) error
	GetRunningJobs() ([]*services.BatchJob, error)
	GetJobsByStatus(status string) ([]*services.BatchJob, error)
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
func (r *batchRepository) Create(job *services.BatchJob) error {
	return r.db.Create(job).Error
}

// GetByID 根据ID获取批量任务
func (r *batchRepository) GetByID(id uint64) (*services.BatchJob, error) {
	var job services.BatchJob
	err := r.db.Where("id = ?", id).First(&job).Error
	return &job, err
}

// GetByUserIDAndID 根据用户ID和任务ID获取批量任务
func (r *batchRepository) GetByUserIDAndID(userID, jobID uint64) (*services.BatchJob, error) {
	var job services.BatchJob
	err := r.db.Where("user_id = ? AND id = ?", userID, jobID).First(&job).Error
	return &job, err
}

// GetByUserID 根据用户ID获取批量任务列表
func (r *batchRepository) GetByUserID(userID uint64, offset, limit int) ([]*services.BatchJob, int64, error) {
	var jobs []*services.BatchJob
	var total int64

	query := r.db.Where("user_id = ?", userID)

	// 获取总数
	if err := query.Model(&services.BatchJob{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&jobs).Error

	return jobs, total, err
}

// Update 更新批量任务
func (r *batchRepository) Update(job *services.BatchJob) error {
	return r.db.Save(job).Error
}

// Delete 删除批量任务
func (r *batchRepository) Delete(jobID uint64) error {
	return r.db.Delete(&services.BatchJob{}, jobID).Error
}

// GetRunningJobs 获取运行中的任务
func (r *batchRepository) GetRunningJobs() ([]*services.BatchJob, error) {
	var jobs []*services.BatchJob
	err := r.db.Where("status = ?", "running").Find(&jobs).Error
	return jobs, err
}

// GetJobsByStatus 根据状态获取任务
func (r *batchRepository) GetJobsByStatus(status string) ([]*services.BatchJob, error) {
	var jobs []*services.BatchJob
	err := r.db.Where("status = ?", status).Find(&jobs).Error
	return jobs, err
}

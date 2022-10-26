package db

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhatinsights/platform-changelog-go/internal/metrics"
	"github.com/redhatinsights/platform-changelog-go/internal/models"
	"gorm.io/gorm"
)

/**
 * GetTimeline returns a timeline of commits and deploys for a service
 */
func GetTimelinesAll(db *gorm.DB, offset int, limit int) ([]models.Timelines, int64, error) {
	callDurationTimer := prometheus.NewTimer(metrics.SqlGetTimelinesAll)
	defer callDurationTimer.ObserveDuration()

	var count int64
	var timelines []models.Timelines

	// Concatanate the timeline fields
	fields := fmt.Sprintf("%s,%s,%s", strings.Join(timelinesFields, ","), strings.Join(commitsFields, ","), strings.Join(deploysFields, ","))

	db = db.Model(models.Timelines{}).Select(fields)

	db.Find(&timelines).Count(&count)
	result := db.Order("Timestamp desc").Limit(limit).Offset(offset).Find(&timelines)

	return timelines, count, result.Error
}

func GetTimelinesByService(db *gorm.DB, service models.Services, offset int, limit int) ([]models.Timelines, int64, error) {
	callDurationTimer := prometheus.NewTimer(metrics.SqlGetTimelinesByService)
	defer callDurationTimer.ObserveDuration()

	var count int64
	var timelines []models.Timelines

	// Concatanate the timeline fields
	fields := fmt.Sprintf("%s,%s,%s", strings.Join(timelinesFields, ","), strings.Join(commitsFields, ","), strings.Join(deploysFields, ","))

	db = db.Model(models.Timelines{}).Select(fields).Where("service_id = ?", service.ID)

	db.Find(&timelines).Count(&count)
	result := db.Order("Timestamp desc").Limit(limit).Offset(offset).Find(&timelines)

	return timelines, count, result.Error
}

func GetTimelineByRef(db *gorm.DB, ref string) (models.Timelines, int64, error) {
	callDurationTimer := prometheus.NewTimer(metrics.SqlGetTimelineByRef)
	defer callDurationTimer.ObserveDuration()

	var timeline models.Timelines

	result := db.Model(models.Timelines{}).Select("*").Where("timelines.ref = ?", ref).Find(&timeline)

	return timeline, result.RowsAffected, result.Error
}

package repo

import (
	"context"
	"time"

	"services/ingest-b/internal/db"

	"gorm.io/gorm"
)

type Filter struct {
	ID1        *string
	ID2        *int
	SensorType *string
	From       *time.Time
	To         *time.Time
}

type Reading struct {
	Value      float64   `json:"value"`
	SensorType string    `json:"sensor_type"`
	ID1        string    `json:"id1"`
	ID2        int       `json:"id2"`
	Timestamp  time.Time `json:"timestamp"`
}

type Store struct {
	DB *gorm.DB
}

func NewStore(d *gorm.DB) *Store { return &Store{DB: d} }

func (s *Store) Insert(ctx context.Context, rs []Reading) error {
	rows := make([]db.SensorRow, 0, len(rs))
	for _, r := range rs {
		rows = append(rows, db.SensorRow{
			Value:      r.Value,
			SensorType: r.SensorType,
			ID1:        r.ID1,
			ID2:        r.ID2,
			TS:         r.Timestamp,
		})
	}
	return s.DB.WithContext(ctx).Create(&rows).Error
}

func (s *Store) applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.ID1 != nil {
		q = q.Where("id1 = ?", *f.ID1)
	}
	if f.ID2 != nil {
		q = q.Where("id2 = ?", *f.ID2)
	}
	if f.SensorType != nil {
		q = q.Where("sensor_type = ?", *f.SensorType)
	}
	if f.From != nil {
		q = q.Where("ts >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("ts <= ?", *f.To)
	}
	return q
}

func (s *Store) Query(ctx context.Context, f Filter, page, limit int) (total int64, out []Reading, err error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	q := s.applyFilter(s.DB.WithContext(ctx).Model(&db.SensorRow{}), f)
	if err = q.Count(&total).Error; err != nil {
		return
	}
	var rows []db.SensorRow
	err = q.Order("ts asc").Limit(limit).Offset((page - 1) * limit).Find(&rows).Error
	if err != nil {
		return
	}
	out = make([]Reading, 0, len(rows))
	for _, r := range rows {
		out = append(out, Reading{
			Value:      r.Value,
			SensorType: r.SensorType,
			ID1:        r.ID1,
			ID2:        r.ID2,
			Timestamp:  r.TS,
		})
	}
	return
}

// Edit supports updating only the value field (simple and safe).
func (s *Store) Edit(ctx context.Context, f Filter, newValue float64) (affected int64, err error) {
	q := s.applyFilter(s.DB.WithContext(ctx).Model(&db.SensorRow{}), f)
	tx := q.Update("value", newValue)
	return tx.RowsAffected, tx.Error
}

func (s *Store) Delete(ctx context.Context, f Filter) (affected int64, err error) {
	q := s.applyFilter(s.DB.WithContext(ctx).Model(&db.SensorRow{}), f)
	tx := q.Delete(&db.SensorRow{})
	return tx.RowsAffected, tx.Error
}

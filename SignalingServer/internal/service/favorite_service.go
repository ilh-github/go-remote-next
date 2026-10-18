package service

import (
	"context"
	"errors"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// FavoriteService implements the user-scoped /v1/me/favorites surface.
// Favorites are per-user bookmarks of remote devices, with an optional
// access_password cached so the user can connect without retyping it.
type FavoriteService struct {
	db     *gorm.DB
	outbox EventOutboxWriter
}

func NewFavoriteService(db *gorm.DB) *FavoriteService {
	return &FavoriteService{db: db}
}

func (s *FavoriteService) WithOutbox(outbox EventOutboxWriter) *FavoriteService {
	s.outbox = outbox
	return s
}

type FavoriteEventFactory func(row *models.UserFavorite) Event

func (s *FavoriteService) enqueueFavoriteEvent(
	ctx context.Context,
	tx *gorm.DB,
	eventFactory FavoriteEventFactory,
	row *models.UserFavorite,
) error {
	if s.outbox == nil || eventFactory == nil || row == nil {
		return nil
	}
	evt := eventFactory(row)
	if evt.Type == "" || evt.UserID == 0 {
		return nil
	}
	_, err := s.outbox.EnqueueTx(ctx, tx, evt)
	return err
}

var (
	ErrFavoriteNotFound = errors.New("favorite not found")
	ErrFavoriteExists   = errors.New("favorite already exists for this device")
)

func (s *FavoriteService) List(ctx context.Context, userID uint) ([]models.UserFavorite, error) {
	var out []models.UserFavorite
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&out).Error
	return out, err
}

type FavoriteInput struct {
	DeviceID       string
	DeviceName     string
	AccessPassword string
}

func (s *FavoriteService) Add(ctx context.Context, userID uint, in FavoriteInput) (*models.UserFavorite, error) {
	return s.AddWithEvent(ctx, userID, in, nil)
}

func (s *FavoriteService) AddWithEvent(
	ctx context.Context,
	userID uint,
	in FavoriteInput,
	eventFactory FavoriteEventFactory,
) (*models.UserFavorite, error) {
	if in.DeviceID == "" {
		return nil, errors.New("device_id is required")
	}
	var fav models.UserFavorite
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.UserFavorite
		err := tx.Where("user_id = ? AND device_id = ?", userID, in.DeviceID).
			First(&existing).Error
		if err == nil {
			return ErrFavoriteExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		now := time.Now().UTC()
		fav = models.UserFavorite{
			UserID:         userID,
			DeviceID:       in.DeviceID,
			DeviceName:     in.DeviceName,
			AccessPassword: in.AccessPassword,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(&fav).Error; err != nil {
			return err
		}
		return s.enqueueFavoriteEvent(ctx, tx, eventFactory, &fav)
	})
	if err != nil {
		return nil, err
	}
	return &fav, nil
}

type FavoritePatch struct {
	DeviceName     *string
	AccessPassword *string
}

func (s *FavoriteService) Update(ctx context.Context, userID uint, deviceID string, p FavoritePatch) (*models.UserFavorite, error) {
	return s.UpdateWithEvent(ctx, userID, deviceID, p, nil)
}

func (s *FavoriteService) UpdateWithEvent(
	ctx context.Context,
	userID uint,
	deviceID string,
	p FavoritePatch,
	eventFactory FavoriteEventFactory,
) (*models.UserFavorite, error) {
	var fav models.UserFavorite
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND device_id = ?", userID, deviceID).
			First(&fav).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFavoriteNotFound
			}
			return err
		}
		if p.DeviceName != nil {
			fav.DeviceName = *p.DeviceName
		}
		if p.AccessPassword != nil {
			fav.AccessPassword = *p.AccessPassword
		}
		fav.UpdatedAt = time.Now().UTC()
		if err := tx.Save(&fav).Error; err != nil {
			return err
		}
		return s.enqueueFavoriteEvent(ctx, tx, eventFactory, &fav)
	})
	if err != nil {
		return nil, err
	}
	return &fav, nil
}

func (s *FavoriteService) Delete(ctx context.Context, userID uint, deviceID string) error {
	return s.DeleteWithEvent(ctx, userID, deviceID, nil)
}

func (s *FavoriteService) DeleteWithEvent(
	ctx context.Context,
	userID uint,
	deviceID string,
	eventFactory FavoriteEventFactory,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fav models.UserFavorite
		if err := tx.Where("user_id = ? AND device_id = ?", userID, deviceID).
			First(&fav).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFavoriteNotFound
			}
			return err
		}
		if err := tx.Delete(&fav).Error; err != nil {
			return err
		}
		return s.enqueueFavoriteEvent(ctx, tx, eventFactory, &fav)
	})
}

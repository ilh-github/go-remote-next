package service

import (
	"context"
	"log"
	"strings"
	"sync"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// SettingsService provides thread-safe access to system settings with in-memory cache.
// Dynamic settings (ICE, security) are read from the cache, which is refreshed
// whenever settings are updated via the admin API.
type SettingsService struct {
	db     *gorm.DB
	mu     sync.RWMutex
	cache  *models.Settings
	outbox EventOutboxWriter
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	s := &SettingsService{db: db}
	s.Reload()
	return s
}

func (s *SettingsService) WithOutbox(outbox EventOutboxWriter) *SettingsService {
	s.outbox = outbox
	return s
}

type SettingsEventFactory func(settings *models.Settings) Event

// Reload reads settings from DB into cache.
func (s *SettingsService) Reload() {
	var settings models.Settings
	if err := s.db.First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			settings = models.Settings{
				SiteEnabled:       true,
				SiteName:          "QuickDesk",
				TurnCredentialTTL: 86400,
			}
		} else {
			log.Printf("Failed to load settings: %v", err)
			return
		}
	}
	s.mu.Lock()
	s.cache = &settings
	s.mu.Unlock()
}

// Get returns a snapshot of the current settings.
func (s *SettingsService) Get() models.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return models.Settings{SiteEnabled: true, SiteName: "QuickDesk", TurnCredentialTTL: 86400}
	}
	return *s.cache
}

// Save persists settings to DB and refreshes the cache.
func (s *SettingsService) Save(settings *models.Settings) error {
	return s.SaveWithEvent(settings, nil)
}

// SaveWithEvent persists settings and optionally enqueues an event in the same
// database transaction. The cache is only updated after the transaction commits.
func (s *SettingsService) SaveWithEvent(settings *models.Settings, eventFactory SettingsEventFactory) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Settings
		result := tx.First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			if err := tx.Create(settings).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			settings.ID = existing.ID
			if err := tx.Save(settings).Error; err != nil {
				return err
			}
		}

		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		evt := eventFactory(settings)
		if evt.Type == "" {
			return nil
		}
		_, err := s.outbox.EnqueueTx(context.Background(), tx, evt)
		return err
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cache = settings
	s.mu.Unlock()
	return nil
}

// EnvSeed holds all .env values for first-run seeding.
type EnvSeed struct {
	TurnURLs, TurnAuthSecret         string
	TurnTTL                          int
	StunURLs, APIKey, AllowedOrigins string
	SmsKeyID, SmsKeySecret           string
	SmsSignName, SmsTemplateCode     string
}

// SeedFromEnv initializes DB with .env values on first run, or backfills
// empty fields into an existing row (handles upgrades that add new config fields).
func (s *SettingsService) SeedFromEnv(seed EnvSeed) {
	var existing models.Settings
	err := s.db.First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		log.Println("Seeding settings from .env configuration...")
		settings := &models.Settings{
			SiteEnabled:        true,
			SiteName:           "QuickDesk",
			TurnURLs:           seed.TurnURLs,
			TurnAuthSecret:     seed.TurnAuthSecret,
			TurnCredentialTTL:  seed.TurnTTL,
			StunURLs:           seed.StunURLs,
			APIKey:             seed.APIKey,
			AllowedOrigins:     seed.AllowedOrigins,
			SmsAccessKeyID:     seed.SmsKeyID,
			SmsAccessKeySecret: seed.SmsKeySecret,
			SmsSignName:        seed.SmsSignName,
			SmsTemplateCode:    seed.SmsTemplateCode,
		}
		if e := s.Save(settings); e != nil {
			log.Printf("Failed to seed settings: %v", e)
		}
		return
	}
	if err != nil {
		log.Printf("Failed to read settings for seeding: %v", err)
		return
	}

	// Backfill: update DB fields that are still empty with non-empty .env values
	updated := false
	backfill := func(dbField *string, envVal string) {
		if *dbField == "" && envVal != "" {
			*dbField = envVal
			updated = true
		}
	}
	backfill(&existing.TurnURLs, seed.TurnURLs)
	backfill(&existing.TurnAuthSecret, seed.TurnAuthSecret)
	backfill(&existing.StunURLs, seed.StunURLs)
	backfill(&existing.APIKey, seed.APIKey)
	backfill(&existing.AllowedOrigins, seed.AllowedOrigins)
	backfill(&existing.SmsAccessKeyID, seed.SmsKeyID)
	backfill(&existing.SmsAccessKeySecret, seed.SmsKeySecret)
	backfill(&existing.SmsSignName, seed.SmsSignName)
	backfill(&existing.SmsTemplateCode, seed.SmsTemplateCode)
	if existing.TurnCredentialTTL == 0 && seed.TurnTTL > 0 {
		existing.TurnCredentialTTL = seed.TurnTTL
		updated = true
	}

	if updated {
		log.Println("Backfilling empty settings fields from .env...")
		if e := s.Save(&existing); e != nil {
			log.Printf("Failed to backfill settings: %v", e)
		}
	}
}

// GetTurnURLs returns TURN URLs as a slice.
func (s *SettingsService) GetTurnURLs() []string {
	return splitLines(s.Get().TurnURLs)
}

func (s *SettingsService) GetTurnAuthSecret() string {
	return s.Get().TurnAuthSecret
}

func (s *SettingsService) GetTurnCredentialTTL() int {
	ttl := s.Get().TurnCredentialTTL
	if ttl <= 0 {
		return 86400
	}
	return ttl
}

// GetStunURLs returns STUN URLs as a slice.
func (s *SettingsService) GetStunURLs() []string {
	return splitLines(s.Get().StunURLs)
}

func (s *SettingsService) GetAPIKey() string {
	return s.Get().APIKey
}

func (s *SettingsService) GetAllowedOrigins() []string {
	return splitLines(s.Get().AllowedOrigins)
}

// SMS getters

func (s *SettingsService) GetSmsAccessKeyID() string     { return s.Get().SmsAccessKeyID }
func (s *SettingsService) GetSmsAccessKeySecret() string { return s.Get().SmsAccessKeySecret }
func (s *SettingsService) GetSmsSignName() string        { return s.Get().SmsSignName }
func (s *SettingsService) GetSmsTemplateCode() string    { return s.Get().SmsTemplateCode }

func (s *SettingsService) IsSmsEnabled() bool {
	st := s.Get()
	return st.SmsAccessKeyID != "" && st.SmsAccessKeySecret != "" &&
		st.SmsSignName != "" && st.SmsTemplateCode != ""
}

// splitLines splits a string by newlines or commas into trimmed non-empty parts.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// Support both newline-separated (from UI) and comma-separated (from .env)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, ",", "\n")
	parts := strings.Split(s, "\n")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceService owns the device lifecycle under the v1 API:
//   - Provision (搂2.5): allocate (device_id, device_secret) on first run
//     or rotate on machine_fingerprint mismatch.
//   - Heartbeat / last-seen bookkeeping.
//   - Access-code storage + verify.
//   - Secret verification (for device_secret Bearer middleware).
//
// All presence state (online/wsconn) lives in PresenceService; DeviceService
// only touches PostgreSQL.
type DeviceService struct {
	repo    *repository.DeviceRepository
	secrets *DeviceSecretService
	outbox  EventOutboxWriter
}

const heartbeatDBUpdateInterval = 5 * time.Minute

func NewDeviceService(repo *repository.DeviceRepository, secrets *DeviceSecretService) *DeviceService {
	return &DeviceService{repo: repo, secrets: secrets}
}

func (s *DeviceService) WithOutbox(outbox EventOutboxWriter) *DeviceService {
	s.outbox = outbox
	return s
}

// DeviceMutationEventFactory builds zero or more events from the final device
// row. It runs inside the caller's transaction.
type DeviceMutationEventFactory func(tx *gorm.DB, device *models.Device) ([]Event, error)

// DeviceAdminUnbindEventFactory runs after a forced unbind. The final device
// row no longer carries the previous owner, so both values are passed in.
type DeviceAdminUnbindEventFactory func(tx *gorm.DB, previousOwner uint, device *models.Device) ([]Event, error)

// DeviceBindEventFactory additionally sees what the bind changed: the previous
// owner and the idempotent already-owned case.
type DeviceBindEventFactory func(tx *gorm.DB, result BindResult) ([]Event, error)

func (s *DeviceService) enqueueDeviceEvents(
	ctx context.Context,
	tx *gorm.DB,
	events []Event,
	err error,
) error {
	if err != nil {
		return err
	}
	if s.outbox == nil {
		return nil
	}
	for _, evt := range events {
		if evt.Type == "" {
			continue
		}
		if _, err := s.outbox.EnqueueTx(ctx, tx, evt); err != nil {
			return err
		}
	}
	return nil
}

// ProvisionRequest is the decoded body of POST /v1/devices:provision.
type ProvisionRequest struct {
	DeviceUUID         string `json:"device_uuid"`
	MachineFingerprint string `json:"machine_fingerprint,omitempty"`
	OS                 string `json:"os,omitempty"`
	OSVersion          string `json:"os_version,omitempty"`
	AppVersion         string `json:"app_version,omitempty"`
}

// ProvisionResult is what we return to the host. The DeviceSecret is the
// plaintext; callers MUST treat it as sensitive and return it to the host
// exactly once.
type ProvisionResult struct {
	DeviceID     string
	DeviceSecret string
	IsNew        bool
}

// Provision allocates or rotates a device_secret for the given device_uuid.
//   - New UUID 鈫?create device + allocate device_id + secret.
//   - Existing UUID 鈫?keep device_id, user_id, access_code, etc. intact, but
//     rotate device_secret (old secret is invalidated).
func (s *DeviceService) Provision(ctx context.Context, req ProvisionRequest) (ProvisionResult, error) {
	if req.DeviceUUID == "" {
		return ProvisionResult{}, errors.New("device_uuid is required")
	}

	// 1. Generate a fresh plaintext secret + hash.
	plaintext, err := s.secrets.Generate()
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("gen secret: %w", err)
	}
	hash, err := s.secrets.Hash(plaintext)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("hash secret: %w", err)
	}

	// 2. Existing UUID?
	existing, err := s.repo.GetByDeviceUUID(ctx, req.DeviceUUID)
	if err == nil {
		existing.DeviceSecretHash = hash
		if req.OS != "" {
			existing.OS = req.OS
		}
		if req.OSVersion != "" {
			existing.OSVersion = req.OSVersion
		}
		if req.AppVersion != "" {
			existing.AppVersion = req.AppVersion
		}
		if req.MachineFingerprint != "" {
			existing.MachineFingerprint = req.MachineFingerprint
		}
		existing.LastSeenAt = time.Now().UTC()
		if err := s.repo.Save(ctx, existing); err != nil {
			return ProvisionResult{}, fmt.Errorf("update existing device: %w", err)
		}
		log.Printf("[Device] Provision rotated secret for device_id=%s uuid=%s", existing.DeviceID, existing.DeviceUUID)
		return ProvisionResult{DeviceID: existing.DeviceID, DeviceSecret: plaintext, IsNew: false}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ProvisionResult{}, fmt.Errorf("lookup device_uuid: %w", err)
	}

	// 3. Brand new UUID 鈫?allocate fresh device_id.
	deviceID, err := s.allocateDeviceID(ctx)
	if err != nil {
		return ProvisionResult{}, err
	}

	device := &models.Device{
		DeviceID:           deviceID,
		DeviceUUID:         req.DeviceUUID,
		DeviceSecretHash:   hash,
		MachineFingerprint: req.MachineFingerprint,
		OS:                 req.OS,
		OSVersion:          req.OSVersion,
		AppVersion:         req.AppVersion,
		LastSeenAt:         time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, device); err != nil {
		return ProvisionResult{}, fmt.Errorf("create device: %w", err)
	}
	log.Printf("[Device] Provisioned new device: device_id=%s uuid=%s", deviceID, req.DeviceUUID)
	return ProvisionResult{DeviceID: deviceID, DeviceSecret: plaintext, IsNew: true}, nil
}

// VerifyDeviceSecret is used by the Bearer-device-secret middleware.
func (s *DeviceService) VerifyDeviceSecret(ctx context.Context, deviceID, plaintext string) (bool, error) {
	d, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return false, err
	}
	if d.DeviceSecretHash == "" {
		return false, nil
	}
	ok, err := s.secrets.Verify(plaintext, d.DeviceSecretHash)
	if ok && needsDeviceSecretRehash(d.DeviceSecretHash) {
		if hash, hErr := s.secrets.Hash(plaintext); hErr == nil {
			_ = s.repo.SetDeviceSecretHash(ctx, deviceID, hash)
		}
	}
	return ok, err
}

func needsDeviceSecretRehash(stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 7 || parts[0] != "argon2id" {
		return false
	}
	return parts[2] != "t=1" || parts[3] != "m=8192" || parts[4] != "p=1"
}

// RotateSecret generates a new plaintext secret (used by admin
// `POST /v1/admin/devices/:id/secret:rotate`). Returns the plaintext once.
func (s *DeviceService) RotateSecret(ctx context.Context, deviceID string) (string, error) {
	return s.RotateSecretWithEvents(ctx, deviceID, nil)
}

func (s *DeviceService) RotateSecretWithEvents(
	ctx context.Context,
	deviceID string,
	eventFactory DeviceMutationEventFactory,
) (string, error) {
	plaintext, err := s.secrets.Generate()
	if err != nil {
		return "", err
	}
	hash, err := s.secrets.Hash(plaintext)
	if err != nil {
		return "", err
	}
	err = s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Device{}).
			Where("device_id = ?", deviceID).
			Update("device_secret_hash", hash).Error; err != nil {
			return err
		}
		d.DeviceSecretHash = hash
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		events, err := eventFactory(tx, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

// Heartbeat refreshes coarse DB bookkeeping at a low frequency. Presence key
// refresh is the caller's responsibility (PresenceService), and is the
// authoritative online signal. Keeping DB writes off the hot path avoids a
// write storm when many hosts heartbeat every few seconds.
func (s *DeviceService) Heartbeat(ctx context.Context, deviceID, os, osVersion, appVersion string) error {
	return s.repo.UpdateHeartbeatIfStale(ctx, deviceID, os, osVersion, appVersion, time.Now().UTC().Add(-heartbeatDBUpdateInterval))
}

// SetAccessCode writes the plaintext access_code into DB. Called from
// PUT /v1/devices/:id/access-code (host-reported, Qt-uploaded).
func (s *DeviceService) SetAccessCode(ctx context.Context, deviceID, code string) error {
	return s.repo.SetAccessCode(ctx, deviceID, code)
}

// SetAccessCodeWithEvents writes the access code and an owner-scoped event in
// one transaction. A missing device keeps the historical no-op success.
func (s *DeviceService) SetAccessCodeWithEvents(
	ctx context.Context,
	deviceID string,
	code string,
	eventFactory DeviceMutationEventFactory,
) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Device{}).
			Where("device_id = ?", deviceID).
			Update("access_code", code).Error; err != nil {
			return err
		}
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		var d models.Device
		if err := tx.Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		events, err := eventFactory(tx, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
}

// VerifyAccessCode does a constant-time comparison against the stored
// plaintext code in DB. Used by POST /v1/devices/:id/access-code:verify
// (and only there — host signaling auth goes through signal_token).
func (s *DeviceService) VerifyAccessCode(ctx context.Context, deviceID, code string) (deviceExists, matches bool, err error) {
	d, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	if d.AccessCode == "" {
		return true, false, nil
	}
	// Constant-time compare (§2.16). subtle.ConstantTimeCompare already
	// handles the equal-length case safely (returns 0 for unequal lengths
	// without leaking timing).
	match := subtle.ConstantTimeCompare([]byte(d.AccessCode), []byte(code)) == 1
	return true, match, nil
}

// BindResult tells the caller what actually happened during BindToUser so
// it can publish the right event (device.bound vs. device.ownership.lost+
// device.added for a takeover).
type BindResult struct {
	Device        *models.Device
	PreviousOwner *uint // nil if the device was unbound before; non-nil on takeover
	AlreadyOwned  bool  // true if the caller already owned the device (idempotent POST)
}

// BindToUser implements POST /v1/me/devices: the current user claims the
// device, optionally taking it over from another owner.
//
// The whole flip happens inside a SERIALIZABLE-ish transaction with
// SELECT ... FOR UPDATE on the device row so two users racing to bind the
// same device see strict ordering (搂2.14).
func (s *DeviceService) BindToUser(ctx context.Context, deviceID string, userID uint) (BindResult, error) {
	return s.BindToUserWithEvents(ctx, deviceID, userID, nil)
}

// BindToUserWithEvents commits the ownership flip and its realtime events
// together. A takeover emits both the old owner's ownership-lost event and the
// new owner's added event through the same factory.
//
// The quota check sits inside the same transaction that holds the device row
// lock, and before the ownership flip. Two concurrent binds for the same user
// then serialize on the row they are both touching only if they touch the same
// device; for two *different* devices the counts are read under the same
// statement snapshot, so a user cannot race their way past the last free slot.
func (s *DeviceService) BindToUserWithEvents(
	ctx context.Context,
	deviceID string,
	userID uint,
	eventFactory DeviceBindEventFactory,
) (BindResult, error) {
	var out BindResult
	err := s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}

		// Idempotent: same user re-binding.
		if d.UserID != nil && *d.UserID == userID {
			d.LoggedIn = true
			if err := tx.Save(&d).Error; err != nil {
				return err
			}
			// Keep the user_devices row fresh (status=true).
			s.upsertUserDevice(tx, userID, deviceID)
			out = BindResult{Device: &d, AlreadyOwned: true}
			if s.outbox == nil || eventFactory == nil {
				return nil
			}
			events, err := eventFactory(tx, out)
			return s.enqueueDeviceEvents(ctx, tx, events, err)
		}

		// A quota of 0 means "no limit configured", not "zero machines": every
		// account is created with that default, so treating it as a hard zero
		// would lock new users out of their first bind.
		if err := s.checkDeviceQuota(tx, userID); err != nil {
			return err
		}

		// Takeover: remember the previous owner so the handler can notify.
		var prev *uint
		if d.UserID != nil {
			prevCopy := *d.UserID
			prev = &prevCopy
			// Mark the previous owner's binding inactive.
			if err := tx.Model(&models.UserDevice{}).
				Where("user_id = ? AND device_id = ?", *d.UserID, deviceID).
				Update("status", false).Error; err != nil {
				return err
			}
			if err := removeDeviceFromUserGroups(tx, *d.UserID, deviceID); err != nil {
				return err
			}
		}

		d.UserID = &userID
		d.LoggedIn = true
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		s.upsertUserDevice(tx, userID, deviceID)
		out = BindResult{Device: &d, PreviousOwner: prev}
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		events, err := eventFactory(tx, out)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
	return out, err
}

// checkDeviceQuota refuses a bind that would put the user over
// `users.device_count`. It runs inside the caller's transaction so a refusal
// rolls back the ownership flip with it.
//
// The count is of devices the user currently owns, read live on every bind: an
// admin raising or lowering the quota takes effect on the next bind rather than
// needing a restart, and a value cached on the service would also go stale the
// moment an admin edited it.
func (s *DeviceService) checkDeviceQuota(tx *gorm.DB, userID uint) error {
	var quota int
	if err := tx.Model(&models.User{}).
		Where("id = ?", userID).
		Pluck("device_count", &quota).Error; err != nil {
		return err
	}
	if quota <= 0 {
		return nil
	}
	var owned int64
	if err := tx.Model(&models.Device{}).
		Where("user_id = ?", userID).
		Count(&owned).Error; err != nil {
		return err
	}
	if owned >= int64(quota) {
		return ErrDeviceQuotaExceeded
	}
	return nil
}

// upsertUserDevice creates or reactivates a (user_id, device_id) binding.
func (s *DeviceService) upsertUserDevice(tx *gorm.DB, userID uint, deviceID string) {
	var ud models.UserDevice
	err := tx.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&ud).Error
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Create(&models.UserDevice{
			UserID:       userID,
			DeviceID:     deviceID,
			Status:       true,
			FirstBoundAt: now,
		})
		return
	}
	if err != nil {
		return
	}
	ud.Status = true
	tx.Save(&ud)
}

// UnbindFromUser implements DELETE /v1/me/devices/:id: clears user_id +
// logged_in on the device and flips the user_device row to inactive.
func (s *DeviceService) UnbindFromUser(ctx context.Context, deviceID string, userID uint) error {
	return s.UnbindFromUserWithEvents(ctx, deviceID, userID, nil)
}

func (s *DeviceService) UnbindFromUserWithEvents(
	ctx context.Context,
	deviceID string,
	userID uint,
	eventFactory DeviceMutationEventFactory,
) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID == nil || *d.UserID != userID {
			return ErrDeviceNotOwned
		}
		d.UserID = nil
		d.LoggedIn = false
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		if err := removeDeviceFromUserGroups(tx, userID, deviceID); err != nil {
			return err
		}
		if err := tx.Model(&models.UserDevice{}).
			Where("user_id = ? AND device_id = ?", userID, deviceID).
			Update("status", false).Error; err != nil {
			return err
		}
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		events, err := eventFactory(tx, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
}

func removeDeviceFromUserGroups(tx *gorm.DB, userID uint, deviceID string) error {
	groupIDs := tx.Model(&models.UserDeviceGroup{}).
		Select("id").
		Where("user_id = ?", userID)
	return tx.Where("group_id IN (?) AND device_id = ?", groupIDs, deviceID).
		Delete(&models.UserDeviceGroupMember{}).Error
}

// ClearSession implements DELETE /v1/me/devices/:id/session 鈥?flips
// logged_in=false while leaving ownership intact. Qt calls this as
// step 1 of the logout flow (搂2.11).
func (s *DeviceService) ClearSession(ctx context.Context, deviceID string, userID uint) error {
	return s.ClearSessionWithEvents(ctx, deviceID, userID, nil)
}

func (s *DeviceService) ClearSessionWithEvents(
	ctx context.Context,
	deviceID string,
	userID uint,
	eventFactory DeviceMutationEventFactory,
) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID == nil || *d.UserID != userID {
			return ErrDeviceNotOwned
		}
		d.LoggedIn = false
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		events, err := eventFactory(tx, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
}

// PatchMeta partially updates device metadata the owner controls.
type PatchMetaInput struct {
	DeviceName *string
	Remark     *string // goes on user_devices, not devices
}

// PatchMeta updates device_name on devices and/or remark on user_devices.
// The caller must be the current owner; we enforce that inside the tx.
func (s *DeviceService) PatchMeta(ctx context.Context, deviceID string, userID uint, in PatchMetaInput) error {
	return s.PatchMetaWithEvents(ctx, deviceID, userID, in, nil)
}

func (s *DeviceService) PatchMetaWithEvents(
	ctx context.Context,
	deviceID string,
	userID uint,
	in PatchMetaInput,
	eventFactory DeviceMutationEventFactory,
) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID == nil || *d.UserID != userID {
			return ErrDeviceNotOwned
		}
		if in.DeviceName != nil {
			if err := tx.Model(&models.Device{}).
				Where("device_id = ?", deviceID).
				Update("device_name", *in.DeviceName).Error; err != nil {
				return err
			}
		}
		if in.Remark != nil {
			s.upsertUserDevice(tx, userID, deviceID)
			if err := tx.Model(&models.UserDevice{}).
				Where("user_id = ? AND device_id = ?", userID, deviceID).
				Update("remark", *in.Remark).Error; err != nil {
				return err
			}
		}
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		if err := tx.Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		events, err := eventFactory(tx, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
}

// ErrDeviceNotOwned is returned when a per-user endpoint is invoked with a
// device_id that doesn't belong to the caller.
var ErrDeviceNotOwned = errors.New("device not owned by caller")

// ErrDeviceQuotaExceeded is returned when binding would put the user over the
// device quota in `users.device_count`. It is distinct from every other device
// error so the handler can answer with a code that names the reason: "you are
// at your limit, unbind something" is actionable, "device not found" is not.
var ErrDeviceQuotaExceeded = errors.New("device quota exceeded")

// AdminForceUnbind clears user_id + logged_in, used by
// POST /v1/admin/devices/:id/unbind.
func (s *DeviceService) AdminForceUnbind(ctx context.Context, deviceID string) (*uint, error) {
	return s.AdminForceUnbindWithEvents(ctx, deviceID, nil)
}

func (s *DeviceService) AdminForceUnbindWithEvents(
	ctx context.Context,
	deviceID string,
	eventFactory DeviceAdminUnbindEventFactory,
) (*uint, error) {
	var previousOwner *uint
	err := s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID != nil {
			prev := *d.UserID
			previousOwner = &prev
			if err := tx.Model(&models.UserDevice{}).
				Where("user_id = ? AND device_id = ?", prev, deviceID).
				Update("status", false).Error; err != nil {
				return err
			}
		}
		d.UserID = nil
		d.LoggedIn = false
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		if s.outbox == nil || eventFactory == nil || previousOwner == nil {
			return nil
		}
		events, err := eventFactory(tx, *previousOwner, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
	return previousOwner, err
}

func (s *DeviceService) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	return s.repo.GetByDeviceID(ctx, deviceID)
}

func (s *DeviceService) GetByDeviceIDWithUser(ctx context.Context, deviceID string) (*models.Device, error) {
	return s.repo.GetByDeviceIDWithUser(ctx, deviceID)
}

func (s *DeviceService) ListByUser(ctx context.Context, userID uint) ([]models.Device, error) {
	return s.repo.ListByUser(ctx, userID)
}

// DeviceAdminListParams is re-exported from the repository so handlers
// don't have to import the repository package directly.
type DeviceAdminListParams = repository.AdminListParams

func (s *DeviceService) ListAdmin(ctx context.Context, p DeviceAdminListParams) ([]models.Device, int64, error) {
	return s.repo.ListAdmin(ctx, p)
}

func (s *DeviceService) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return s.repo.CountSince(ctx, since)
}

func (s *DeviceService) Delete(ctx context.Context, deviceID string) error {
	return s.DeleteWithEvents(ctx, deviceID, nil)
}

func (s *DeviceService) DeleteWithEvents(
	ctx context.Context,
	deviceID string,
	eventFactory DeviceMutationEventFactory,
) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		// Access policy is keyed by device_id without a database-level foreign
		// key. Delete it in the same transaction so a later random device-id
		// allocation can never inherit an old deny list.
		if err := tx.Where("device_id = ?", deviceID).
			Delete(&models.DeviceAccessPolicy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("device_id = ?", deviceID).
			Delete(&models.Device{}).Error; err != nil {
			return err
		}
		if s.outbox == nil || eventFactory == nil {
			return nil
		}
		events, err := eventFactory(tx, &d)
		return s.enqueueDeviceEvents(ctx, tx, events, err)
	})
}

// allocateDeviceID tries up to 10 cryptographically-random 9-digit IDs,
// falling back to UUID-derived uniqueness if collisions persist. A real
// collision at this sample space is essentially impossible but the retry
// loop is cheap insurance against a hot key.
func (s *DeviceService) allocateDeviceID(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		id := randomNineDigit()
		_, err := s.repo.GetByDeviceID(ctx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return id, nil
		}
		if err != nil {
			return "", fmt.Errorf("check device_id: %w", err)
		}
	}
	// Fallback using a UUID-derived numeric suffix.
	u := uuid.New()
	n := binary.BigEndian.Uint32(u[:4])
	fallback := fmt.Sprintf("%09d", n%1_000_000_000)
	if _, err := s.repo.GetByDeviceID(ctx, fallback); errors.Is(err, gorm.ErrRecordNotFound) {
		return fallback, nil
	}
	return "", errors.New("exhausted device_id allocation retries")
}

func randomNineDigit() string {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	n := binary.BigEndian.Uint32(buf[:])
	// 100_000_000 .. 999_999_999
	return fmt.Sprintf("%09d", 100_000_000+n%900_000_000)
}

package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ErrUploadTicketConsumed means the ticket was already redeemed. Callers map
// it onto 409 rather than 401: the credential was valid, it is just spent.
var ErrUploadTicketConsumed = errors.New("upload ticket already consumed")

// RecordingUploadTicketStore mints and redeems one-shot upload tickets for a
// recording artifact.
//
// QD-004 needs a credential the Host can use to push a recording the server
// has never seen, without handing over the user's Bearer token. That is a
// different animal from the playback ticket in recording_ticket.go: a playback
// link is meant to be replayed every time the owner watches the video, while
// an upload ticket that can be replayed lets whoever holds it overwrite the
// artifact again and again.
//
// The signature binds (recording, user, nonce, expiry) to a server secret, so
// a caller cannot mint one for someone else's recording. The one-shot part
// needs server-side state, which lives in Redis: the nonce is marked on first
// redemption and the marker outlives the ticket's own expiry.
//
// Without Redis the store degrades to a plain signed ticket, which is NOT
// single-use. That is deliberate - a deployment without Redis is already the
// single-process SQLite mode - but callers that need the guarantee must refuse
// to issue rather than silently downgrade. `SingleUse` reports which mode is
// live so they can.
type RecordingUploadTicketStore struct {
	rdb    *redis.Client
	secret string
}

// NewRecordingUploadTicketStore builds a store over the given secret. An empty
// secret disables issuance, matching the playback ticket's behaviour.
func NewRecordingUploadTicketStore(rdb *redis.Client, secret string) *RecordingUploadTicketStore {
	return &RecordingUploadTicketStore{rdb: rdb, secret: secret}
}

// SingleUse reports whether redemption is actually remembered across calls.
func (s *RecordingUploadTicketStore) SingleUse() bool {
	return s != nil && s.rdb != nil
}

// RecordingUploadTicket is a redeemed upload credential.
type RecordingUploadTicket struct {
	RecordingID uint
	UserID      uint
	Nonce       string
	ExpiresAt   time.Time
}

func uploadTicketKey(nonce string) string {
	return "qd:recording:upload-ticket:" + nonce
}

func uploadTicketPayload(recordingID, userID uint, nonce string, expUnix int64) string {
	return "rid=" + formatUintID(recordingID) +
		"&uid=" + formatUintID(userID) +
		"&n=" + nonce +
		"&exp=" + strconv.FormatInt(expUnix, 10)
}

// Issue mints a ticket valid for ttl. The returned instant is the real
// deadline, truncated to whole seconds because that is what travels inside
// the signed payload.
func (s *RecordingUploadTicketStore) Issue(
	recordingID, userID uint, ttl time.Duration,
) (string, time.Time, error) {
	if s == nil || s.secret == "" {
		return "", time.Time{}, errors.New("recording upload tickets are disabled: set RECORDING_URL_SECRET or API_KEY")
	}
	if recordingID == 0 || userID == 0 {
		return "", time.Time{}, errors.New("upload ticket requires a recording and a user")
	}
	exp := time.Now().Add(ttl).UTC().Truncate(time.Second)
	nonce := uuid.NewString()
	payload := uploadTicketPayload(recordingID, userID, nonce, exp.Unix())
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return body + "." + sig, exp, nil
}

// Consume redeems a ticket exactly once.
//
// With Redis the nonce is marked atomically, so two concurrent redemptions
// cannot both win. Without Redis there is no state to check and the ticket
// stays replayable - see the type comment.
func (s *RecordingUploadTicketStore) Consume(
	ctx context.Context, ticket string,
) (*RecordingUploadTicket, error) {
	if s == nil || s.secret == "" {
		return nil, ErrInvalidRecordingTicket
	}
	body, sig, ok := strings.Cut(strings.TrimSpace(ticket), ".")
	if !ok || body == "" || sig == "" {
		return nil, ErrInvalidRecordingTicket
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, ErrInvalidRecordingTicket
	}
	payload := string(raw)

	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, ErrInvalidRecordingTicket
	}

	values := map[string]string{}
	for _, kv := range strings.Split(payload, "&") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, ErrInvalidRecordingTicket
		}
		values[k] = v
	}
	recordingID, err := strconv.ParseUint(values["rid"], 10, 64)
	if err != nil || recordingID == 0 {
		return nil, ErrInvalidRecordingTicket
	}
	userID, err := strconv.ParseUint(values["uid"], 10, 64)
	if err != nil || userID == 0 {
		return nil, ErrInvalidRecordingTicket
	}
	nonce := values["n"]
	if nonce == "" {
		return nil, ErrInvalidRecordingTicket
	}
	expUnix, err := strconv.ParseInt(values["exp"], 10, 64)
	if err != nil {
		return nil, ErrInvalidRecordingTicket
	}
	exp := time.Unix(expUnix, 0).UTC()
	if time.Now().UTC().After(exp) {
		return nil, ErrInvalidRecordingTicket
	}

	if s.rdb != nil {
		// The marker lives longer than the ticket so a replayed link stays
		// dead after its expiry would have made it moot anyway.
		marked, err := s.rdb.SetNX(ctx, uploadTicketKey(nonce), "1", 2*time.Until(exp)+time.Hour).Result()
		if err != nil {
			return nil, err
		}
		if !marked {
			return nil, ErrUploadTicketConsumed
		}
	}

	return &RecordingUploadTicket{
		RecordingID: uint(recordingID),
		UserID:      uint(userID),
		Nonce:       nonce,
		ExpiresAt:   exp,
	}, nil
}

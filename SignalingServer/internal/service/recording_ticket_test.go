package service

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRecordingTicketRoundTrip(t *testing.T) {
	ticket, exp := SignRecordingTicket("secret", 42, 7, 5*time.Minute)
	if !exp.After(time.Now().UTC()) {
		t.Fatalf("ticket expiry %v is not in the future", exp)
	}
	got, err := VerifyRecordingTicket("secret", ticket)
	if err != nil {
		t.Fatalf("VerifyRecordingTicket() error = %v", err)
	}
	if got.RecordingID != 42 || got.UserID != 7 {
		t.Fatalf("ticket = (recording %d, user %d), want (42, 7)", got.RecordingID, got.UserID)
	}
	if !got.ExpiresAt.Equal(exp) {
		t.Fatalf("expiry = %v, want %v", got.ExpiresAt, exp)
	}
}

func TestRecordingTicketRejectsWrongSecret(t *testing.T) {
	ticket, _ := SignRecordingTicket("secret", 42, 7, time.Minute)
	if _, err := VerifyRecordingTicket("other-secret", ticket); !errors.Is(err, ErrInvalidRecordingTicket) {
		t.Fatalf("error = %v, want ErrInvalidRecordingTicket", err)
	}
}

func TestRecordingTicketRejectsExpired(t *testing.T) {
	// Sign with a negative TTL so the ticket is already expired.
	ticket, _ := SignRecordingTicket("secret", 42, 7, -time.Minute)
	if _, err := VerifyRecordingTicket("secret", ticket); !errors.Is(err, ErrInvalidRecordingTicket) {
		t.Fatalf("error = %v, want ErrInvalidRecordingTicket", err)
	}
}

func TestRecordingTicketRejectsTampering(t *testing.T) {
	ticket, _ := SignRecordingTicket("secret", 42, 7, time.Minute)
	body, sig, _ := strings.Cut(ticket, ".")

	// Swapping the body for another recording must not keep the old signature.
	other, _ := SignRecordingTicket("secret", 43, 7, time.Minute)
	otherBody, _, _ := strings.Cut(other, ".")
	if _, err := VerifyRecordingTicket("secret", otherBody+"."+sig); !errors.Is(err, ErrInvalidRecordingTicket) {
		t.Fatalf("error = %v, want ErrInvalidRecordingTicket", err)
	}

	malformed := []string{"", "nodot", body + ".", "." + sig, "!!!." + sig}
	for _, in := range malformed {
		if _, err := VerifyRecordingTicket("secret", in); !errors.Is(err, ErrInvalidRecordingTicket) {
			t.Fatalf("VerifyRecordingTicket(%q) error = %v, want ErrInvalidRecordingTicket", in, err)
		}
	}
}

package invite

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/Sidd231/4yaar/backend/internal/modules/email"
	"github.com/google/uuid"
)

var (
	ErrInviteNotFound = errors.New("invite not found")
	ErrInviteExpired  = errors.New("invite has expired")
	ErrInviteUsed     = errors.New("invite has already been used")
)

// VerifyInviteResponse represents the result of validating an invitation token.
type VerifyInviteResponse struct {
	Valid     bool      `json:"valid"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service defines invite business workflows.
type Service interface {
	CreateInvite(ctx context.Context, inviterID uuid.UUID, inviterName, email string) (*models.Invite, error)
	VerifyInvite(ctx context.Context, token uuid.UUID) (*VerifyInviteResponse, error)
}

type service struct {
	repo         Repository
	emailService email.Service
}

// NewService instantiates an invite service.
func NewService(repo Repository, emailService email.Service) Service {
	return &service{
		repo:         repo,
		emailService: emailService,
	}
}

// CreateInvite issues a single-use 48h invitation token and triggers an invitation email.
func (s *service) CreateInvite(ctx context.Context, inviterID uuid.UUID, inviterName, toEmail string) (*models.Invite, error) {
	now := time.Now()
	token := uuid.New()

	invite := &models.Invite{
		ID:        uuid.New(),
		Email:     toEmail,
		Token:     token,
		InvitedBy: inviterID,
		ExpiresAt: now.Add(48 * time.Hour),
		CreatedAt: now,
	}

	if err := s.repo.CreateInvite(ctx, invite); err != nil {
		return nil, fmt.Errorf("failed to save invite: %w", err)
	}

	// Send invite email asynchronously
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.emailService.SendInviteEmail(bgCtx, toEmail, token.String(), inviterName); err != nil {
			slog.Warn("Failed to send invite email", "to", toEmail, "error", err)
		}
	}()

	return invite, nil
}

// VerifyInvite checks if an invitation token is authentic, unused, and not expired.
func (s *service) VerifyInvite(ctx context.Context, token uuid.UUID) (*VerifyInviteResponse, error) {
	invite, err := s.repo.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup invite: %w", err)
	}
	if invite == nil {
		return nil, ErrInviteNotFound
	}
	if invite.IsExpired() {
		return nil, ErrInviteExpired
	}
	if invite.IsUsed() {
		return nil, ErrInviteUsed
	}

	return &VerifyInviteResponse{
		Valid:     true,
		Email:     invite.Email,
		ExpiresAt: invite.ExpiresAt,
	}, nil
}

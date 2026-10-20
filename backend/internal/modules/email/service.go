package email

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Sidd231/4yaar/backend/internal/config"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// Service defines email dispatching operations via Amazon SES.
type Service interface {
	SendInviteEmail(ctx context.Context, toEmail, inviteToken, inviterName string) error
	SendWelcomeEmail(ctx context.Context, toEmail, userName string) error
}

type sesService struct {
	client    *ses.Client
	fromEmail string
	baseURL   string
}

// NewEmailService initializes an Amazon SES email client.
func NewEmailService(cfg *config.Config) (Service, error) {
	var optFns []func(*awsconfig.LoadOptions) error
	optFns = append(optFns, awsconfig.WithRegion(cfg.AWSRegion))

	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config for SES: %w", err)
	}

	client := ses.NewFromConfig(awsCfg)

	return &sesService{
		client:    client,
		fromEmail: cfg.SESFromEmail,
		baseURL:   cfg.AppBaseURL,
	}, nil
}

// SendInviteEmail sends an invitation link with the given single-use token.
func (s *sesService) SendInviteEmail(ctx context.Context, toEmail, inviteToken, inviterName string) error {
	inviteURL := fmt.Sprintf("%s/register?token=%s", s.baseURL, inviteToken)
	subject, bodyHTML, err := RenderInviteEmail(inviteURL, inviterName)
	if err != nil {
		return err
	}

	return s.send(ctx, toEmail, subject, bodyHTML)
}

// SendWelcomeEmail dispatches a welcome email once user registration completes.
func (s *sesService) SendWelcomeEmail(ctx context.Context, toEmail, userName string) error {
	loginURL := fmt.Sprintf("%s/login", s.baseURL)
	subject, bodyHTML, err := RenderWelcomeEmail(userName, loginURL)
	if err != nil {
		return err
	}

	return s.send(ctx, toEmail, subject, bodyHTML)
}

// send issues the SES SendEmail API call.
func (s *sesService) send(ctx context.Context, toEmail, subject, bodyHTML string) error {
	input := &ses.SendEmailInput{
		Source: aws.String(s.fromEmail),
		Destination: &sestypes.Destination{
			ToAddresses: []string{toEmail},
		},
		Message: &sestypes.Message{
			Subject: &sestypes.Content{
				Data:    aws.String(subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &sestypes.Body{
				Html: &sestypes.Content{
					Data:    aws.String(bodyHTML),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		slog.Error("Failed to send email via SES",
			"to", toEmail,
			"subject", subject,
			"error", err,
		)
		return fmt.Errorf("ses send email failed: %w", err)
	}

	slog.Info("Email sent successfully via SES", "to", toEmail, "subject", subject)
	return nil
}

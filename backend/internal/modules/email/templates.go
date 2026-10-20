package email

import (
	"bytes"
	"fmt"
	"html/template"
)

const inviteEmailTmpl = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0f172a; color: #f8fafc; margin: 0; padding: 40px 20px; }
    .card { max-width: 520px; margin: 0 auto; background-color: #1e293b; border-radius: 12px; padding: 36px; border: 1px solid #334155; }
    h1 { color: #38bdf8; font-size: 24px; margin-top: 0; }
    p { font-size: 15px; line-height: 1.6; color: #cbd5e1; }
    .btn { display: inline-block; background: linear-gradient(135deg, #0284c7, #2563eb); color: #ffffff !important; text-decoration: none; padding: 12px 28px; border-radius: 8px; font-weight: 600; margin-top: 20px; }
    .footer { font-size: 12px; color: #64748b; margin-top: 30px; border-top: 1px solid #334155; padding-top: 15px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>You're invited to 4Yaar</h1>
    <p>Hello,</p>
    <p><strong>{{.InviterName}}</strong> has invited you to join the exclusive, members-only collaboration space on <strong>4Yaar</strong>.</p>
    <p>This invitation link is single-use and will expire in 48 hours.</p>
    <p style="text-align: center;">
      <a href="{{.InviteURL}}" class="btn" target="_blank">Accept Invitation & Register</a>
    </p>
    <p>Or paste this link into your browser:</p>
    <p style="word-break: break-all; color: #38bdf8; font-size: 13px;">{{.InviteURL}}</p>
    <div class="footer">
      If you did not expect this invitation, you can safely ignore this email.
    </div>
  </div>
</body>
</html>
`

const welcomeEmailTmpl = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0f172a; color: #f8fafc; margin: 0; padding: 40px 20px; }
    .card { max-width: 520px; margin: 0 auto; background-color: #1e293b; border-radius: 12px; padding: 36px; border: 1px solid #334155; }
    h1 { color: #38bdf8; font-size: 24px; margin-top: 0; }
    p { font-size: 15px; line-height: 1.6; color: #cbd5e1; }
    .btn { display: inline-block; background: linear-gradient(135deg, #0284c7, #2563eb); color: #ffffff !important; text-decoration: none; padding: 12px 28px; border-radius: 8px; font-weight: 600; margin-top: 20px; }
    .footer { font-size: 12px; color: #64748b; margin-top: 30px; border-top: 1px solid #334155; padding-top: 15px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>Welcome to 4Yaar, {{.UserName}}!</h1>
    <p>Your membership account has been created and verified successfully.</p>
    <p>You now have full access to real-time chat rooms, encrypted file storage, and instant group video calls.</p>
    <p style="text-align: center;">
      <a href="{{.LoginURL}}" class="btn" target="_blank">Launch 4Yaar</a>
    </p>
    <div class="footer">
      4Yaar — Secure, Private Real-Time Collaboration.
    </div>
  </div>
</body>
</html>
`

// RenderInviteEmail generates the subject line and HTML body for an invitation email.
func RenderInviteEmail(inviteURL, inviterName string) (string, string, error) {
	tmpl, err := template.New("invite").Parse(inviteEmailTmpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse invite email template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		InviteURL   string
		InviterName string
	}{
		InviteURL:   inviteURL,
		InviterName: inviterName,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("failed to render invite email template: %w", err)
	}

	subject := fmt.Sprintf("%s invited you to join 4Yaar", inviterName)
	return subject, buf.String(), nil
}

// RenderWelcomeEmail generates the subject line and HTML body for a welcome email.
func RenderWelcomeEmail(userName, loginURL string) (string, string, error) {
	tmpl, err := template.New("welcome").Parse(welcomeEmailTmpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse welcome email template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		UserName string
		LoginURL string
	}{
		UserName: userName,
		LoginURL: loginURL,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("failed to render welcome email template: %w", err)
	}

	subject := "Welcome to 4Yaar"
	return subject, buf.String(), nil
}

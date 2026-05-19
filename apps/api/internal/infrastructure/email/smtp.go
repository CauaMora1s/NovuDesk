package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/novudesk/novudesk/config"
	"github.com/novudesk/novudesk/internal/domain/email"
)

type SMTPSender struct {
	cfg       config.SMTPConfig
	templates *template.Template
}

func NewSMTPSender(cfg config.SMTPConfig) (*SMTPSender, error) {
	// Parse embedded templates at startup — fail fast.
	tmpl, err := template.New("email").Parse(baseTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse email templates: %w", err)
	}
	return &SMTPSender{cfg: cfg, templates: tmpl}, nil
}

func (s *SMTPSender) Send(_ context.Context, msg email.Message) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromAddress),
		"To":           strings.Join(msg.To, ", "),
		"Subject":      msg.Subject,
		"MIME-Version": "1.0",
		"Content-Type": `multipart/alternative; boundary="boundary"`,
	}

	var buf bytes.Buffer
	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	buf.WriteString("\r\n")
	fmt.Fprintf(&buf, "--boundary\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n", msg.Text)
	fmt.Fprintf(&buf, "--boundary\r\nContent-Type: text/html; charset=utf-8\r\n\r\n%s\r\n--boundary--", msg.HTML)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	// For dev (MailHog) we skip TLS; for prod, use TLS when port is 465/587.
	if s.cfg.Port == 465 {
		return s.sendTLS(addr, auth, msg.To, buf.Bytes())
	}

	return smtp.SendMail(addr, auth, s.cfg.FromAddress, msg.To, buf.Bytes())
}

func (s *SMTPSender) sendTLS(addr string, auth smtp.Auth, to []string, body []byte) error {
	host := strings.Split(addr, ":")[0]
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(s.cfg.FromAddress); err != nil {
		return err
	}
	for _, r := range to {
		if err := client.Rcpt(r); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	w.Write(body)
	return w.Close()
}

// baseTemplate is a minimal responsive email shell.
const baseTemplate = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:sans-serif;background:#f9fafb;padding:32px 0;margin:0">
  <table width="600" cellpadding="0" cellspacing="0" style="margin:0 auto;background:#fff;border-radius:8px;padding:32px">
    <tr><td>{{.Body}}</td></tr>
  </table>
</body>
</html>`

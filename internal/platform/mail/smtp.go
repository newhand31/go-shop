// Package mail sends the shop's emails over SMTP, with the standard library only.
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Config says where the mail server is and who the shop writes to. There is no customer email address yet (no accounts),
// so To is one configured address: the shop owner's, or a test mailbox.
type Config struct {
	Host string
	Port string
	From string
	To   string
}

// SMTP sends order confirmations through one SMTP server. It uses no authentication: it is meant for a relay on the
// private network or a test server. If the server offers STARTTLS, it is used.
type SMTP struct {
	cfg     Config
	timeout time.Duration
}

// NewSMTP builds a sender. Empty From and To get placeholders on the reserved .test domain.
func NewSMTP(cfg Config) *SMTP {
	if cfg.From == "" {
		cfg.From = "shop@go-shop.test"
	}
	if cfg.To == "" {
		cfg.To = "customer@go-shop.test"
	}
	return &SMTP{cfg: cfg, timeout: 10 * time.Second}
}

// SendOrderConfirmation mails one confirmation. It fails when the server is unreachable, and the outbox worker then
// retries the row later: that is the whole point of the outbox, so this method does not retry by itself.
func (s *SMTP) SendOrderConfirmation(ctx context.Context, orderID, totalTWD int) error {
	subject := fmt.Sprintf("Order %d confirmed", orderID)
	body := fmt.Sprintf("Thank you for your order.\r\n\r\nOrder number: %d\r\nTotal: NT$%d\r\n", orderID, totalTWD)
	return s.send(ctx, subject, body)
}

func (s *SMTP) send(ctx context.Context, subject, body string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	// The deadline covers the whole conversation, not only the dial: a server that accepts and then goes silent must not
	// hold the outbox worker forever.
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp greeting from %s: %w", addr, err)
	}
	defer func() { _ = c.Close() }()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if err := c.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := c.Rcpt(s.cfg.To); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	msg := strings.Join([]string{
		"From: " + s.cfg.From,
		"To: " + s.cfg.To,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp end of message: %w", err)
	}
	return c.Quit()
}

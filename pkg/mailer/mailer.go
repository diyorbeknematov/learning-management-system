package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const (
	// implicitTLSPort is the port where the connection is encrypted from the
	// first byte; on other ports STARTTLS is used when the server offers it.
	implicitTLSPort = 465
	dialTimeout     = 10 * time.Second
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the sender address, e.g. "noreply@example.com".
	From string
}

type Mailer struct {
	cfg Config
}

func New(cfg Config) (*Mailer, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" {
		return nil, errors.New("mailer: host, port and from are required")
	}

	if hasLineBreak(cfg.From) {
		return nil, errors.New("mailer: invalid from address")
	}

	return &Mailer{cfg: cfg}, nil
}

// Send sends a plain text (UTF-8) email. The context limits the whole
// delivery, including connecting.
func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	if to == "" || hasLineBreak(to) || hasLineBreak(subject) {
		return errors.New("mailer: invalid recipient or subject")
	}

	client, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer: authentication: %w", err)
		}
	}

	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("mailer: sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: data: %w", err)
	}

	if _, err := writer.Write(m.message(to, subject, body)); err != nil {
		return fmt.Errorf("mailer: write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("mailer: send message: %w", err)
	}

	return client.Quit()
}

func (m *Mailer) connect(ctx context.Context) (*smtp.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	tlsConfig := &tls.Config{ServerName: m.cfg.Host}

	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("mailer: connect to %s: %w", addr, err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if m.cfg.Port == implicitTLSPort {
		conn = tls.Client(conn, tlsConfig)
	}

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()

		return nil, fmt.Errorf("mailer: smtp handshake: %w", err)
	}

	if m.cfg.Port != implicitTLSPort {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				client.Close()

				return nil, fmt.Errorf("mailer: starttls: %w", err)
			}
		}
	}

	// the deadline above only protects connecting and the handshake
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	return client, nil
}

func (m *Mailer) message(to, subject, body string) []byte {
	headers := []string{
		"From: " + m.cfg.From,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}

	// SMTP requires CRLF line endings
	body = strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n")
}

func hasLineBreak(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

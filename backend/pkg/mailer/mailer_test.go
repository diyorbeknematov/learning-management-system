package mailer_test

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/pkg/mailer"
	"github.com/stretchr/testify/require"
)

type received struct {
	from string
	to   []string
	data string
}

// fakeServer is a small SMTP server: enough of the protocol for the mailer to
// connect, log in and deliver a message. It has no TLS, so it advertises no
// STARTTLS and the mailer sends plain text to it.
type fakeServer struct {
	listener net.Listener

	username   string // when set the server requires AUTH PLAIN with these
	password   string
	rejectRcpt bool

	mu       sync.Mutex
	messages []received
}

// newFakeServer starts the server; the options run before it accepts
// connections, so they need no locking.
func newFakeServer(t *testing.T, options ...func(*fakeServer)) *fakeServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &fakeServer{listener: listener}

	for _, option := range options {
		option(server)
	}

	go server.serve()

	t.Cleanup(func() { _ = listener.Close() })

	return server
}

func withAuth(username, password string) func(*fakeServer) {
	return func(s *fakeServer) {
		s.username = username
		s.password = password
	}
}

func withRejectedRecipients(s *fakeServer) {
	s.rejectRcpt = true
}

func (s *fakeServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *fakeServer) config() mailer.Config {
	return mailer.Config{
		Host:     "127.0.0.1",
		Port:     s.port(),
		Username: s.username,
		Password: s.password,
		From:     "noreply@lms.local",
	}
}

func (s *fakeServer) received() []received {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]received(nil), s.messages...)
}

func (s *fakeServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		go s.handle(conn)
	}
}

func (s *fakeServer) handle(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	reply := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	var current received
	authenticated := s.username == ""

	reply("220 fake ready")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "EHLO"):
			if s.username != "" {
				reply("250-fake")
				reply("250 AUTH PLAIN")
			} else {
				reply("250 fake")
			}
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			payload, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(line[len("AUTH PLAIN"):]))
			parts := strings.Split(string(payload), "\x00")

			if len(parts) == 3 && parts[1] == s.username && parts[2] == s.password {
				authenticated = true
				reply("235 ok")
			} else {
				reply("535 bad credentials")
			}
		case strings.HasPrefix(upper, "MAIL FROM:"):
			if !authenticated {
				reply("530 authentication required")
				continue
			}

			current = received{from: extractAddress(line)}
			reply("250 ok")
		case strings.HasPrefix(upper, "RCPT TO:"):
			if s.rejectRcpt {
				reply("550 no such user")
				continue
			}

			current.to = append(current.to, extractAddress(line))
			reply("250 ok")
		case upper == "DATA":
			reply("354 go ahead")

			var data strings.Builder

			for {
				part, err := reader.ReadString('\n')
				if err != nil {
					return
				}

				if part == ".\r\n" {
					break
				}

				data.WriteString(strings.TrimPrefix(part, "."))
			}

			current.data = data.String()

			s.mu.Lock()
			s.messages = append(s.messages, current)
			s.mu.Unlock()

			reply("250 queued")
		case upper == "QUIT":
			reply("221 bye")

			return
		default:
			reply("250 ok")
		}
	}
}

func extractAddress(line string) string {
	start := strings.Index(line, "<")
	end := strings.Index(line, ">")

	if start < 0 || end < start {
		return ""
	}

	return line[start+1 : end]
}

func parse(t *testing.T, raw string) (*mail.Message, string) {
	t.Helper()

	message, err := mail.ReadMessage(strings.NewReader(raw))
	require.NoError(t, err)

	body, err := io.ReadAll(message.Body)
	require.NoError(t, err)

	return message, string(body)
}

func TestNew_Validation(t *testing.T) {
	valid := mailer.Config{Host: "smtp.example.com", Port: 587, From: "noreply@example.com"}

	_, err := mailer.New(valid)
	require.NoError(t, err)

	for name, mutate := range map[string]func(*mailer.Config){
		"no host":          func(c *mailer.Config) { c.Host = "" },
		"no port":          func(c *mailer.Config) { c.Port = 0 },
		"no from":          func(c *mailer.Config) { c.From = "" },
		"line break, from": func(c *mailer.Config) { c.From = "a@b.c\r\nBcc: x@y.z" },
	} {
		cfg := valid
		mutate(&cfg)

		_, err := mailer.New(cfg)
		require.Error(t, err, name)
	}
}

func TestSend(t *testing.T) {
	server := newFakeServer(t)

	m, err := mailer.New(server.config())
	require.NoError(t, err)

	err = m.Send(
		context.Background(),
		"student@example.com",
		"Parolni tiklash — LMS",
		"Salom!\nHavola: http://localhost:3000/reset?token=abc\nO'g'il bola, 15 daqiqa.",
	)
	require.NoError(t, err)

	messages := server.received()
	require.Len(t, messages, 1)
	require.Equal(t, "noreply@lms.local", messages[0].from)
	require.Equal(t, []string{"student@example.com"}, messages[0].to)

	message, body := parse(t, messages[0].data)

	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, "Parolni tiklash — LMS", subject)

	require.Equal(t, "student@example.com", message.Header.Get("To"))
	require.Equal(t, "noreply@lms.local", message.Header.Get("From"))
	require.Equal(t, "1.0", message.Header.Get("MIME-Version"))
	require.Contains(t, message.Header.Get("Content-Type"), "charset=UTF-8")

	_, err = message.Header.Date()
	require.NoError(t, err)

	require.Equal(t, "Salom!\r\nHavola: http://localhost:3000/reset?token=abc\r\nO'g'il bola, 15 daqiqa.\r\n", body)
}

func TestSend_ConvertsLineEndings(t *testing.T) {
	server := newFakeServer(t)

	m, err := mailer.New(server.config())
	require.NoError(t, err)

	require.NoError(t, m.Send(context.Background(), "a@example.com", "s", "one\ntwo\r\nthree"))

	_, body := parse(t, server.received()[0].data)

	require.Equal(t, "one\r\ntwo\r\nthree\r\n", body)
	require.NotContains(t, strings.ReplaceAll(body, "\r\n", ""), "\n")
}

func TestSend_WithAuth(t *testing.T) {
	server := newFakeServer(t, withAuth("user", "secret"))

	m, err := mailer.New(server.config())
	require.NoError(t, err)

	require.NoError(t, m.Send(context.Background(), "a@example.com", "s", "b"))
	require.Len(t, server.received(), 1)
}

func TestSend_WrongPassword(t *testing.T) {
	server := newFakeServer(t, withAuth("user", "secret"))

	cfg := server.config()
	cfg.Password = "wrong"

	m, err := mailer.New(cfg)
	require.NoError(t, err)

	err = m.Send(context.Background(), "a@example.com", "s", "b")

	require.Error(t, err)
	require.Contains(t, err.Error(), "authentication")
	require.Empty(t, server.received())
}

func TestSend_RecipientRejected(t *testing.T) {
	server := newFakeServer(t, withRejectedRecipients)

	m, err := mailer.New(server.config())
	require.NoError(t, err)

	err = m.Send(context.Background(), "nobody@example.com", "s", "b")

	require.Error(t, err)
	require.Contains(t, err.Error(), "recipient")
	require.Empty(t, server.received())
}

func TestSend_RejectsHeaderInjection(t *testing.T) {
	server := newFakeServer(t)

	m, err := mailer.New(server.config())
	require.NoError(t, err)

	ctx := context.Background()

	require.Error(t, m.Send(ctx, "a@example.com\r\nBcc: evil@example.com", "s", "b"))
	require.Error(t, m.Send(ctx, "a@example.com", "hello\r\nBcc: evil@example.com", "b"))
	require.Error(t, m.Send(ctx, "", "s", "b"))
	require.Empty(t, server.received())
}

func TestSend_ConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	m, err := mailer.New(mailer.Config{Host: "127.0.0.1", Port: port, From: "noreply@lms.local"})
	require.NoError(t, err)

	err = m.Send(context.Background(), "a@example.com", "s", "b")

	require.Error(t, err)
	require.Contains(t, err.Error(), "connect to 127.0.0.1:"+strconv.Itoa(port))
}

func TestSend_ContextCanceled(t *testing.T) {
	server := newFakeServer(t)

	m, err := mailer.New(server.config())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Error(t, m.Send(ctx, "a@example.com", "s", "b"))
	require.Empty(t, server.received())
}

func TestSend_ServerThatNeverAnswers(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			// accepts the connection but never sends the greeting
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	m, err := mailer.New(mailer.Config{
		Host: "127.0.0.1",
		Port: listener.Addr().(*net.TCPAddr).Port,
		From: "noreply@lms.local",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	started := time.Now()

	require.Error(t, m.Send(ctx, "a@example.com", "s", "b"))
	require.Less(t, time.Since(started), 3*time.Second)
}

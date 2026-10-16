package alert

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMTPAlreadyCanceled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := NewSmtpChannel("test", nil, SmtpConfig{Host: listener.Addr().String()})
	result := make(chan error, 1)
	go func() { result <- ch.SendProtoMsg(ctx, "test") }()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled context did not stop dialing")
	}
	require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(50*time.Millisecond)))
	conn, err := listener.Accept()
	if conn != nil {
		conn.Close()
		t.Fatal("dialed despite already canceled context")
	}
	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	assert.True(t, netErr.Timeout())
}

func TestSMTPCancelTLSHandshake(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			ctx, cancel := smtpTestContext(mode)
			defer cancel()
			ch := NewSmtpChannel("test", nil, SmtpConfig{Host: listener.Addr().String()})
			result := make(chan error, 1)
			go func() { result <- ch.SendProtoMsg(ctx, "test") }()
			require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second)))
			conn, err := listener.Accept()
			require.NoError(t, err)
			defer conn.Close()
			// Consume ClientHello, but never complete the TLS handshake.
			require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
			_, err = conn.Read(make([]byte, 4096))
			require.NoError(t, err)
			if mode == "cancel" {
				cancel()
			}
			assertSMTPContextResult(t, ctx, result)
		})
	}
}

func TestSMTPCancelSession(t *testing.T) {
	for _, stage := range []string{"greeting", "EHLO", "AUTH", "MAIL", "RCPT", "DATA", "write", "data reply", "QUIT"} {
		for _, mode := range []string{"cancel", "deadline"} {
			t.Run(stage+"/"+mode, func(t *testing.T) {
				client, server := smtpTLSPipe(t)
				ready, release := make(chan struct{}), make(chan struct{})
				defer close(release)
				serverResult := make(chan error, 1)
				go func() { serverResult <- serveTestSMTP(server, stage, ready, release) }()
				ctx, cancel := smtpTestContext(mode)
				defer cancel()
				ch := NewSmtpChannel("test", nil, SmtpConfig{
					Host: "localhost:465", From: "sender@example.com", To: []string{"recipient@example.com"}, Password: "test",
				})
				result := make(chan error, 1)
				go func() { result <- ch.sendOnConn(ctx, client, strings.Repeat("body", 8192)) }()
				select {
				case <-ready:
				case err := <-serverResult:
					t.Fatalf("server did not reach %s: %v", stage, err)
				case <-time.After(time.Second):
					t.Fatalf("server did not reach %s", stage)
				}
				if mode == "cancel" {
					cancel()
				}
				assertSMTPContextResult(t, ctx, result)
			})
		}
	}
}

func TestSMTPSessionSuccess(t *testing.T) {
	client, server := smtpTLSPipe(t)
	serverResult := make(chan error, 1)
	go func() { serverResult <- serveTestSMTP(server, "", nil, nil) }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ch := NewSmtpChannel("test", nil, SmtpConfig{
		Host: "localhost:465", From: "sender@example.com", To: []string{"recipient@example.com"}, Password: "test",
	})
	require.NoError(t, ch.sendOnConn(ctx, client, "Subject: test\r\n\r\nbody\r\n"))
	require.NoError(t, <-serverResult)
}

func TestSMTPServerError(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	go func() {
		io.WriteString(server, "554 service unavailable\r\n")
		server.Close()
	}()
	ch := NewSmtpChannel("test", nil, SmtpConfig{Host: "localhost:465"})
	err := ch.sendOnConn(context.Background(), client, "test")
	var smtpErr *textproto.Error
	require.ErrorAs(t, err, &smtpErr)
	assert.Equal(t, 554, smtpErr.Code)
	assert.NotErrorIs(t, err, context.Canceled)
}

func smtpTestContext(mode string) (context.Context, context.CancelFunc) {
	if mode == "deadline" {
		return context.WithTimeout(context.Background(), 250*time.Millisecond)
	}
	return context.WithCancel(context.Background())
}

func assertSMTPContextResult(t *testing.T, ctx context.Context, result <-chan error) {
	t.Helper()
	select {
	case err := <-result:
		require.Error(t, ctx.Err(), "SMTP returned before cancellation")
		require.ErrorIs(t, err, ctx.Err())
	case <-time.After(time.Second):
		t.Fatal("SMTP did not return promptly on cancellation")
	}
}

// Use a trusted local certificate and in-memory sockets; no real mail is sent.
func smtpTLSPipe(t *testing.T) (*tls.Conn, *tls.Conn) {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1), DNSNames: []string{"localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, pub, key)
	require.NoError(t, err)
	parsed, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	a, b := net.Pipe()
	t.Cleanup(func() { a.Close(); b.Close() })
	client := tls.Client(a, &tls.Config{RootCAs: roots, ServerName: "localhost"})
	server := tls.Server(b, &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- server.HandshakeContext(ctx) }()
	require.NoError(t, client.HandshakeContext(ctx))
	require.NoError(t, <-result)
	return client, server
}

func serveTestSMTP(conn net.Conn, stall string, ready chan<- struct{}, release <-chan struct{}) error {
	text := textproto.NewConn(conn)
	pause := func(stage string) bool {
		if stall != stage {
			return false
		}
		close(ready)
		<-release
		return true
	}
	if pause("greeting") {
		return nil
	}
	if err := text.PrintfLine("220 localhost SMTP"); err != nil {
		return err
	}
	for _, step := range []struct{ command, response string }{
		{"EHLO", "250-localhost\r\n250 AUTH PLAIN"},
		{"AUTH", "235 authenticated"},
		{"MAIL", "250 sender accepted"},
		{"RCPT", "250 recipient accepted"},
		{"DATA", "354 send data"},
	} {
		line, err := text.ReadLine()
		if err != nil {
			return err
		}
		if !strings.HasPrefix(line, step.command+" ") && line != step.command {
			return fmt.Errorf("expected %s, got %q", step.command, line)
		}
		if pause(step.command) {
			return nil
		}
		if err := text.PrintfLine("%s", step.response); err != nil {
			return err
		}
	}
	if pause("write") {
		return nil
	}
	body, err := text.ReadDotBytes()
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), "body") {
		return fmt.Errorf("message body missing")
	}
	if pause("data reply") {
		return nil
	}
	if err := text.PrintfLine("250 queued"); err != nil {
		return err
	}
	line, err := text.ReadLine()
	if err != nil {
		return err
	}
	if line != "QUIT" {
		return fmt.Errorf("expected QUIT, got %q", line)
	}
	if pause("QUIT") {
		return nil
	}
	if err := text.PrintfLine("221 bye"); err != nil {
		return err
	}
	// Drain the client's TLS close-notify so Quit can close gracefully.
	_, err = io.Copy(io.Discard, conn)
	return err
}

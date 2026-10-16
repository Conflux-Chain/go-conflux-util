package alert

import (
	"context"
	"crypto/tls"
	stderr "errors"
	"net"
	"net/smtp"

	"github.com/pkg/errors"
)

var (
	_ Channel = (*SmtpChannel)(nil)
)

type SmtpConfig struct {
	Host     string   // SMTP endpoint and port
	From     string   // Sender address
	To       []string // Receipt addresses
	Password string   // SMTP password
}

// SmtpChannel represents a SMTP email notification channel
type SmtpChannel struct {
	Formatter Formatter  // Formatter is used to format the notification message
	ID        string     // ID is the identifier of the channel
	Config    SmtpConfig // Config contains the configuration for the SMTP server
}

// NewSmtpChannel creates a new SMTP channel with the given ID, formatter, and configuration
func NewSmtpChannel(chID string, fmtter Formatter, conf SmtpConfig) *SmtpChannel {
	return &SmtpChannel{ID: chID, Formatter: fmtter, Config: conf}
}

// Name returns the ID of the channel
func (c *SmtpChannel) Name() string {
	return c.ID
}

// Type returns the type of the channel, which is SMTP
func (c *SmtpChannel) Type() ChannelType {
	return ChannelTypeSMTP
}

// Send sends a notification using the SMTP channel
func (c *SmtpChannel) Send(ctx context.Context, note *Notification) error {
	// Format the notification message
	msg, err := c.Formatter.Format(note)
	if err != nil {
		return errors.WithMessage(err, "failed to format alert msg")
	}

	// Send the formatted message
	return c.SendProtoMsg(ctx, msg)
}

// SendProtoMsg sends protocol message using the SMTP channel.
// Callers should provide a context with a deadline to bound the entire session.
func (c *SmtpChannel) SendProtoMsg(ctx context.Context, msg string) error {
	dialer := tls.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", c.Config.Host)
	if err != nil {
		return errors.WithMessage(err, "failed to dial smtp server")
	}

	return c.sendOnConn(ctx, conn, msg)
}

// sendOnConn owns conn and watches cancellation before reading the SMTP greeting.
func (c *SmtpChannel) sendOnConn(ctx context.Context, conn net.Conn, msg string) (err error) {
	// Close the underlying socket on cancellation: tls.Conn.Close can wait for
	// a close-notify write, delaying interruption of a blocked SMTP read.
	socket := conn
	if tlsConn, ok := conn.(*tls.Conn); ok {
		socket = tlsConn.NetConn()
	}
	stop := context.AfterFunc(ctx, func() { socket.Close() })
	defer func() {
		stop()
		socket.Close()
		conn.Close()
		if err != nil && ctx.Err() != nil {
			err = stderr.Join(err, ctx.Err())
		}
	}()

	host, _, err := net.SplitHostPort(c.Config.Host)
	if err != nil {
		return errors.WithMessage(err, "invalid smtp address")
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return errors.WithMessage(err, "failed to initialize smtp client")
	}

	// Authenticate with the SMTP server if it supports authentication.
	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", c.Config.From, c.Config.Password, host)
		if err := client.Auth(auth); err != nil {
			return errors.WithMessage(err, "failed to authenticate smtp server")
		}
	}

	if err := client.Mail(c.Config.From); err != nil {
		return errors.WithMessage(err, "failed to set sender")
	}
	for _, addr := range c.Config.To {
		if err := client.Rcpt(addr); err != nil {
			return errors.WithMessagef(err, "failed to add recipient %s", addr)
		}
	}

	w, err := client.Data()
	if err != nil {
		return errors.WithMessage(err, "failed to open data writer")
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return errors.WithMessage(err, "failed to write data")
	}
	if err := w.Close(); err != nil {
		return errors.WithMessage(err, "failed to close data writer")
	}
	if err := client.Quit(); err != nil {
		return errors.WithMessage(err, "failed to quit smtp session")
	}
	return nil
}

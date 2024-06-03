package alert

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/smtp"
	"time"

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
	return c.send(ctx, msg)
}

// SendRaw sends raw protocol message.
func (c *SmtpChannel) SendRaw(ctx context.Context, content interface{}) error {
	msg, ok := content.(string)
	if !ok {
		return ErrInvalidContentType
	}

	return c.send(ctx, msg)
}

// send sends a message using the SMTP channel
func (c *SmtpChannel) send(ctx context.Context, msg string) error {
	// Dial the SMTP server
	client, err := c.dial(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to dial smtp server")
	}

	// Close the client when done
	defer client.Close()

	// Authenticate with the SMTP server if it supports authentication
	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", c.Config.From, c.Config.Password, c.Config.Host)
		if err := c.doWithDeadlineCheck(ctx, func() error { return client.Auth(auth) }); err != nil {
			return errors.WithMessage(err, "failed to authenticate smtp server")
		}
	}

	// Set the sender
	if err := c.doWithDeadlineCheck(ctx, func() error { return client.Mail(c.Config.From) }); err != nil {
		return errors.WithMessage(err, "failed to set sender")
	}

	// Add the recipients
	for _, addr := range c.Config.To {
		if err := c.doWithDeadlineCheck(ctx, func() error { return client.Rcpt(addr) }); err != nil {
			return errors.WithMessagef(err, "failed to add recipient %s", addr)
		}
	}

	// Write the message
	var w io.WriteCloser
	if err := c.doWithDeadlineCheck(ctx, func() error { w, err = client.Data(); return err }); err != nil {
		return errors.WithMessage(err, "failed to open data writer")
	}

	if err := c.doWithDeadlineCheck(ctx, func() error { _, err = w.Write([]byte(msg)); return err }); err != nil {
		return errors.WithMessage(err, "failed to write data")
	}

	// Close the writer
	if err := w.Close(); err != nil {
		return errors.WithMessage(err, "failed to close data writer")
	}

	// Quit the SMTP session
	if err := c.doWithDeadlineCheck(ctx, func() error { return client.Quit() }); err != nil {
		return errors.WithMessage(err, "failed to quit smtp session")
	}

	return nil
}

// doWithDeadlineCheck checks if the deadline has been exceeded before performing an operation
func (c *SmtpChannel) doWithDeadlineCheck(ctx context.Context, op func() error) error {
	if deadlineExceeded(ctx) {
		return context.DeadlineExceeded
	}

	if err := op(); err != nil {
		return err
	}

	return nil
}

// deadlineExceeded checks if the deadline of the context has been exceeded
func deadlineExceeded(ctx context.Context) bool {
	if _, ok := ctx.Deadline(); ok {
		select {
		case <-ctx.Done():
			return true
		default:
		}
	}
	return false
}

// dial dials the SMTP server and returns a new SMTP client
func (c *SmtpChannel) dial(ctx context.Context) (*smtp.Client, error) {
	var dialer *net.Dialer

	// Check if a deadline has been set
	deadline, ok := ctx.Deadline()
	if ok { // deadline set?
		timeout := time.Until(deadline)
		if timeout < 0 { // timeout exceeded
			return nil, context.DeadlineExceeded
		}
		dialer = &net.Dialer{Timeout: timeout}
	} else {
		dialer = new(net.Dialer)
	}

	// Dial the SMTP server
	conn, err := tls.DialWithDialer(dialer, "tcp", c.Config.Host, nil)
	if err != nil {
		return nil, err
	}

	// Create a new SMTP client
	return smtp.NewClient(conn, c.Config.Host)
}

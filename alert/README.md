# Alerting Utilities

This package integrates several popular alerting functions, and provides default templates to send notifications.

## Supported Channels

There are several channels supported to send notifications, and developers could configure any one or several channels in program.

- Telegram
- Dingtalk
- SMTP
- Pagerduty
- Flashduty

## Initialize Channel

Before sending any message to notification channels, the client should create a channel robot. To construct a channel robot programmatically:

```go
// Construct a notification channel imperatively
var notifyCh alert.Channel

// DingTalk Channel
notifyCh = alert.NewDingTalkChannel(...)
// or PagerDuty Channel
notifyCh = alert.NewPagerDutyChannel(...)
// or Smtp email Channel
notifyCh = alert.NewSmtpChannel(...)
// or Telegram Channel
notifyCh = alert.NewTelegramChannel(...)
```

Alternatively, you can initialize the alert channels from configuration file or environment variables, which is recommended.

```go
// Initialize the alert channels from configurations loaded by viper
alert.MustInitFromViper()
// After initialization, you can retrieve the notification channel using a unique channel ID
notifyCh := alert.DefaultManager().Channel(chID)
```

## Send Notification

Once the channel is initialized, you can send a notification message through the channel:

SMTP, FlashDuty and DingTalk use the supplied context for cancellation and do not add
a default request timeout. Set a deadline when calling them directly; the example
below allows three seconds. The logrus alert hook sets its own `SendTimeout`.
SMTP cancellation covers TCP connection establishment, the TLS handshake, and
the entire SMTP session; errors remain identifiable with `errors.Is` against
`context.Canceled` or `context.DeadlineExceeded`.
Webhook URLs must point directly to the final endpoint; redirects are rejected.

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

notifyCh.Send(ctx, &alert.Notification{
    Title:    "Alert testing",
    Severity: alert.SeverityLow,
    Content: `This is a test notification message`,
})
```

## Hook to Logrus

`Alert` can be integrated with [log](../log/README.md) module, so as to send alerting message when `warning` or `error` logs occurred. Generally, developers could initialize `log` and `alert` via [config](../config/README.md).

package alert

import (
	"strings"

	"github.com/mcuadros/go-defaults"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var (
	ErrInvalidNotification = errors.New("invalid notification")
)

func ErrChannelTypeNotSupported(chType string) error {
	return errors.Errorf("channel type %s not supported", chType)
}

func ErrChannelNotFound(ch string) error {
	return errors.Errorf("channel %s not found", ch)
}

func parseAlertChannel(chID string, chmap map[string]interface{}, tags []string) (Channel, error) {
	cht, ok := chmap["platform"].(string)
	if !ok {
		return nil, ErrChannelTypeNotSupported(cht)
	}

	switch ChannelType(cht) {
	case ChannelTypeDingTalk:
		var dtconf DingTalkConfig
		if err := decodeChannelConfig(chmap, &dtconf); err != nil {
			return nil, err
		}

		fmt, err := newDingtalkMsgFormatter(dtconf.MsgType, tags)
		if err != nil {
			return nil, err
		}

		return NewDingTalkChannel(chID, fmt, dtconf), nil
	case ChannelTypeTelegram:
		if toStr, ok := chmap["atusers"].(string); ok {
			mentions := strings.Split(toStr, ",")
			chmap["atusers"] = mentions
		}

		var tgconf TelegramConfig
		if err := decodeChannelConfig(chmap, &tgconf); err != nil {
			return nil, err
		}

		fmt, err := NewTelegramMarkdownFormatter(tags, tgconf.AtUsers)
		if err != nil {
			return nil, err
		}

		return NewTelegramChannel(chID, fmt, tgconf)
	case ChannelTypeSMTP:
		if toStr, ok := chmap["to"].(string); ok {
			recipients := strings.Split(toStr, ",")
			chmap["to"] = recipients
		}

		var smtpconf SmtpConfig
		if err := decodeChannelConfig(chmap, &smtpconf); err != nil {
			return nil, err
		}

		fmt, err := NewSmtpHtmlFormatter(smtpconf, tags)
		if err != nil {
			return nil, err
		}

		return NewSmtpChannel(chID, fmt, smtpconf), nil
	case ChannelTypePagerDuty:
		var pdconf PagerDutyConfig
		if err := decodeChannelConfig(chmap, &pdconf); err != nil {
			return nil, err
		}

		return NewPagerDutyChannel(chID, tags, pdconf), nil

	// NOTE: add more channel types support here if needed
	default:
		return nil, ErrChannelTypeNotSupported(cht)
	}
}

func decodeChannelConfig(chmap map[string]interface{}, valPtr interface{}) error {
	defaults.SetDefaults(valPtr)
	decoderConfig := mapstructure.DecoderConfig{
		TagName: "json",
		Result:  valPtr,
	}

	// Create a new Decoder instance
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return errors.WithMessage(err, "failed to new mapstructure decoder")
	}

	// Decode the map into the struct
	if err := decoder.Decode(chmap); err != nil {
		return errors.WithMessage(err, "failed to decode mapstructure")
	}

	return nil
}

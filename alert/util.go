package alert

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func ErrChannelTypeNotSupported(chType string) error {
	return errors.Errorf("channel type %s not supported", chType)
}

func ErrChannelNotFound(ch string) error {
	return errors.Errorf("channel %s not found", ch)
}

func parseAlertChannel(chID string, chmap map[string]interface{}, fmt Formatter) (Channel, error) {
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

		return NewDingTalkChannel(chID, fmt, dtconf), nil
	case ChannelTypeTelegram:
		var tgconf TelegramConfig
		if err := decodeChannelConfig(chmap, &tgconf); err != nil {
			return nil, err
		}

		return NewTelegramChannel(chID, fmt, tgconf)

	// NOTE: add more channel types support here if needed
	default:
		return nil, ErrChannelTypeNotSupported(cht)
	}
}

func decodeChannelConfig(chmap map[string]interface{}, valPtr interface{}) error {
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

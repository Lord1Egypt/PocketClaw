package channels

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"

	"github.com/sipeed/picoclaw/pkg/config"
)

// toChannelHashes fingerprints every enabled channel's runtime-relevant
// configuration so a reload can tell which channels actually changed.
//
// The settings half of the input is built from the channel's own raw bytes
// rather than from marshalling the channels map. config.Channel serializes from
// its raw bytes until something decodes it and from the decoded struct
// afterwards, and the two shapes hash differently — so hashing the same config
// twice used to produce two different answers and would have restarted every
// live channel. The common fields are read from the struct, where they always
// live, so they are decode-state independent by construction.
func toChannelHashes(cfg *config.Config) map[string]string {
	result := make(map[string]string)
	if cfg == nil {
		return result
	}

	for name, bc := range cfg.Channels {
		if bc == nil || !bc.Enabled {
			continue
		}

		value := make(map[string]any)
		if !bc.SettingsIsEmpty() {
			if err := json.Unmarshal(bc.Settings, &value); err != nil {
				log.Printf("[manager_channel] failed to unmarshal channel %s config: %v", name, err)
				continue
			}
		}
		// Carried explicitly: config.Channel keeps these beside Settings rather
		// than inside it, so a hash built only from the settings payload cannot
		// see them. Leaving them out made a Typing Indicator change — and every
		// other common field — invisible to the reconcile, so saving one never
		// restarted the channel.
		value["enabled"] = bc.Enabled
		value["type"] = bc.Type
		value["typing"] = bc.Typing
		value["placeholder"] = bc.Placeholder
		value["allow_from"] = bc.AllowFrom
		value["reasoning_channel_id"] = bc.ReasoningChannelID
		value["group_trigger"] = bc.GroupTrigger

		hiddenValues(name, value, bc)

		valueBytes, err := json.Marshal(value)
		if err != nil {
			log.Printf("[manager_channel] failed to marshal channel %s config: %v", name, err)
			continue
		}
		hash := md5.Sum(valueBytes)
		result[name] = hex.EncodeToString(hash[:])
	}

	return result
}

// hiddenValues re-introduces the channel's credentials into the map used to
// compute the reconcile hash.
//
// This used to be a hand-maintained switch over channel names, and every
// channel missing from it — weixin, vk, pico_client — had token changes that
// the reconcile could not see, so saving a new token never restarted the
// channel. Walking the decoded settings struct instead covers every channel,
// including ones added later, and cannot fall out of date.
func hiddenValues(_ string, value map[string]any, ch *config.Channel) {
	decoded, err := ch.GetDecoded()
	if err != nil {
		return
	}
	injectSecretFingerprints(value, decoded)
}

func compareChannels(old, news map[string]string) (added, removed []string) {
	for key, newHash := range news {
		if oldHash, ok := old[key]; ok {
			if newHash != oldHash {
				removed = append(removed, key)
				added = append(added, key)
			}
		} else {
			added = append(added, key)
		}
	}
	for key := range old {
		if _, ok := news[key]; !ok {
			removed = append(removed, key)
		}
	}
	return added, removed
}

func toChannelConfig(cfg *config.Config, list []string) (*config.ChannelsConfig, error) {
	result := make(config.ChannelsConfig)
	for _, name := range list {
		bc, ok := cfg.Channels[name]
		if !ok || !bc.Enabled {
			continue
		}
		result[name] = bc
	}
	return &result, nil
}

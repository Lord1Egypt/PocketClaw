package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// The one Telegram owner contract, enforced where every Core writer passes.
//
// PC-DEF-028. NewTelegramChannel has always required exactly one positive
// numeric owner, but nothing on the way in agreed: the web console treated any
// non-empty allow_from as configured and offered "anyone" otherwise, so it
// would happily persist zero owners, several, or a username. Core then refused
// the channel at startup with "telegram requires exactly one paired numeric
// owner" while the console went on showing the channel as configured.
//
// The rule lives here rather than in the UI. Frontend validation is for
// feedback; this is the boundary.

// telegramOwnerErrors reports why a Telegram channel's owner list is invalid.
//
// A disabled channel is not held to the rule -- turning Telegram off must not
// require first repairing it -- and a channel that is absent has no owner to
// check.
func telegramOwnerErrors(bc *config.Channel) []string {
	if bc == nil || !bc.Enabled {
		return nil
	}

	owners := make([]string, 0, len(bc.AllowFrom))
	for _, raw := range bc.AllowFrom {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			owners = append(owners, trimmed)
		}
	}

	switch {
	case len(owners) == 0:
		return []string{"channel \"telegram\" is enabled but has no owner: " +
			"allow_from must contain exactly one numeric Telegram user ID"}
	case len(owners) > 1:
		// Deliberately not resolved by picking one. Which of several owners
		// was intended is the user's decision, and guessing it would hand
		// somebody's agent to whichever entry happened to sort first.
		return []string{fmt.Sprintf(
			"channel %q has %d owners: allow_from must contain exactly one "+
				"numeric Telegram user ID, so remove the others",
			"telegram", len(owners))}
	}

	owner, err := strconv.ParseInt(owners[0], 10, 64)
	if err != nil {
		return []string{fmt.Sprintf(
			"channel %q owner %q is not a numeric Telegram user ID",
			"telegram", owners[0])}
	}
	if owner <= 0 {
		return []string{fmt.Sprintf(
			"channel %q owner must be a positive numeric Telegram user ID", "telegram")}
	}
	return nil
}

// telegramSemanticKey is a canonical view of the Telegram configuration.
//
// Built from decoded fields, never from serialized bytes: a whole-config PUT
// round-trips every value and may reorder keys or change formatting without
// changing anything the product means. Treating that as an edit would make an
// unrelated settings save fail on a legacy Telegram configuration the user was
// not touching.
func telegramSemanticKey(cfg *config.Config) string {
	if cfg == nil || cfg.Channels == nil {
		return "absent"
	}
	bc := cfg.Channels.GetByType(config.ChannelTelegram)
	if bc == nil {
		return "absent"
	}

	owners := make([]string, 0, len(bc.AllowFrom))
	for _, raw := range bc.AllowFrom {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			owners = append(owners, trimmed)
		}
	}

	// Owner order is not meaning: [a b] and [b a] are the same configuration
	// and neither is valid anyway.
	sortStrings(owners)

	key := fmt.Sprintf("enabled=%t;owners=%s", bc.Enabled, strings.Join(owners, ","))

	decoded, err := bc.GetDecoded()
	if err != nil {
		return key + ";settings=undecodable"
	}
	settings, ok := decoded.(*config.TelegramSettings)
	if !ok || settings == nil {
		return key + ";settings=absent"
	}
	// The token itself is never part of the key material that gets compared as
	// a value: only whether one is present, so a changed token still reads as
	// an edit without the secret entering this path.
	return fmt.Sprintf("%s;token_set=%t;base_url=%s;proxy=%s",
		key,
		strings.TrimSpace(settings.Token.String()) != "",
		strings.TrimSpace(settings.BaseURL),
		strings.TrimSpace(settings.Proxy),
	)
}

// telegramSubtreeChanged reports whether a save alters Telegram at all.
func telegramSubtreeChanged(existing, incoming *config.Config) bool {
	return telegramSemanticKey(existing) != telegramSemanticKey(incoming)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

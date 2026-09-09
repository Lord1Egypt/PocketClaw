package config

// Legacy identities for PocketClaw's managed realtime channel.
//
// LEGACY READ-ONLY MIGRATION. These are the values an installation created
// before the Zero-Pico channel migration carries in config.json and
// .security.yml. They are recognised so an existing install keeps working on
// its first start after upgrade, and they are rewritten to the canonical names
// once. Nothing in this build writes them: there is no dual-write, no alias and
// no downgrade copy.
//
// They live here, together, so the compatibility surface is one file rather
// than a scatter of string literals that a later sweep would have to
// distinguish from live identity by reading each one.
const (
	// LegacyChannelPocketClaw is the pre-migration type and config key for the
	// managed realtime channel.
	LegacyChannelPocketClaw = "pico"

	// LegacyChannelPocketClawClient is the pre-migration type and config key
	// for the client half of that channel.
	LegacyChannelPocketClawClient = "pico_client"

	// LegacyPocketClawOwnerPrincipal is the pre-migration owner label found in
	// the managed channel's allow_from list.
	LegacyPocketClawOwnerPrincipal = "pico-user"
)

// legacyChannelNames maps each legacy channel identity to its canonical name.
//
// One table, used by both the config migration and the guards that prove no
// writer emits a legacy value.
var legacyChannelNames = map[string]string{
	LegacyChannelPocketClaw:       ChannelPocketClaw,
	LegacyChannelPocketClawClient: ChannelPocketClawClient,
}

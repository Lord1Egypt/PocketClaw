package config

// The managed realtime channel's HTTP surface.
//
// Three packages need these: the channel registers and parses them, the web
// console proxies them, and the middleware recognises them for logging and
// origin checks. They are derived from the channel identity and declared here,
// where that identity already lives, because the alternative is the same
// strings written out in three places that a rename can move one of.
//
// pkg/logger cannot import this package — config imports logger — so it carries
// its own copy, marked as such, and a test pins the two together.
const (
	// RealtimeRoutePrefix is the namespace every realtime route sits under.
	RealtimeRoutePrefix = "/" + ChannelPocketClaw + "/"

	// RealtimeWebSocketPath is the socket a console or app client connects to.
	RealtimeWebSocketPath = RealtimeRoutePrefix + "ws"

	// RealtimeMediaPrefix is where an attachment is served from.
	RealtimeMediaPrefix = RealtimeRoutePrefix + "media/"

	// RealtimeAPIPrefix is the console's management API for this channel.
	RealtimeAPIPrefix = "/api/" + ChannelPocketClaw + "/"
)

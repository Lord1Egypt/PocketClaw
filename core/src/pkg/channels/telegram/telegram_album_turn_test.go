package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
)

//  11. A seven-image album is one inbound turn, not seven, and the images keep
//     the order Telegram assigned them.
func TestSevenImageAlbumBecomesOneInboundTurn(t *testing.T) {
	messageBus, ch := newMediaGroupTestChannel(20 * time.Millisecond)

	base := testMediaGroupMessage("album-seven")
	for i := 1; i <= 7; i++ {
		msg := base
		msg.MessageID = i
		if i == 1 {
			msg.Caption = "describe these"
		}
		require.NoError(t, ch.handleMessage(context.Background(), &msg))
	}

	var inbound bus.InboundMessage
	select {
	case inbound = <-messageBus.InboundChan():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the album turn")
	}

	assert.Equal(t, "1", inbound.Context.MessageID,
		"the album must be anchored on its first message")
	assert.Equal(t, "describe these", inbound.Content,
		"the caption must survive aggregation")

	// Exactly one turn: seven separate turns is the failure this guards.
	select {
	case extra := <-messageBus.InboundChan():
		t.Fatalf("the album produced more than one turn: %#v", extra)
	case <-time.After(150 * time.Millisecond):
	}
}

// Arrivals spread across the collection window must still land in one turn:
// each new part restarts the debounce rather than splitting the album.
func TestStaggeredAlbumPartsStayInOneTurn(t *testing.T) {
	messageBus, ch := newMediaGroupTestChannel(60 * time.Millisecond)

	base := testMediaGroupMessage("album-staggered")
	for i := 1; i <= 5; i++ {
		msg := base
		msg.MessageID = i
		if i == 1 {
			msg.Caption = "staggered album"
		}
		require.NoError(t, ch.handleMessage(context.Background(), &msg))
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case inbound := <-messageBus.InboundChan():
		assert.Equal(t, "1", inbound.Context.MessageID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the staggered album turn")
	}

	select {
	case extra := <-messageBus.InboundChan():
		t.Fatalf("a staggered album was split across turns: %#v", extra)
	case <-time.After(150 * time.Millisecond):
	}
}

//  12. A later unrelated message is its own turn and is never absorbed into a
//     pending album.
func TestUnrelatedMessageIsNotAbsorbedIntoAPendingAlbum(t *testing.T) {
	messageBus, ch := newMediaGroupTestChannel(time.Hour)

	album := testMediaGroupMessage("album-open")
	album.MessageID = 1
	album.Caption = "album caption"
	require.NoError(t, ch.handleMessage(context.Background(), &album))

	// The album is still collecting; a plain text message must not join it.
	standalone := testMediaGroupMessage("")
	standalone.MessageID = 99
	standalone.Text = "an unrelated question"
	require.NoError(t, ch.handleMessage(context.Background(), &standalone))

	select {
	case inbound := <-messageBus.InboundChan():
		assert.Equal(t, "99", inbound.Context.MessageID,
			"the standalone message must be delivered as its own turn")
		assert.Equal(t, "an unrelated question", inbound.Content)
	case <-time.After(time.Second):
		t.Fatal("the unrelated message was swallowed by the pending album")
	}

	ch.mediaGroupMu.Lock()
	group := ch.mediaGroups["456:album-open"]
	pending := 0
	if group != nil {
		pending = len(group.messages)
	}
	ch.mediaGroupMu.Unlock()
	require.Equal(t, 1, pending,
		"the unrelated message must not have been buffered into the album")
}

// A different album in the same chat is a separate turn.
func TestTwoAlbumsInOneChatAreSeparateTurns(t *testing.T) {
	messageBus, ch := newMediaGroupTestChannel(20 * time.Millisecond)

	for _, groupID := range []string{"album-one", "album-two"} {
		base := testMediaGroupMessage(groupID)
		for i := 1; i <= 2; i++ {
			msg := base
			msg.MessageID = i
			if i == 1 {
				msg.Caption = groupID
			}
			require.NoError(t, ch.handleMessage(context.Background(), &msg))
		}
	}

	seen := map[string]bool{}
	for range 2 {
		select {
		case inbound := <-messageBus.InboundChan():
			seen[inbound.Content] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out; only saw %v", seen)
		}
	}
	assert.True(t, seen["album-one"], "the first album must produce its own turn")
	assert.True(t, seen["album-two"], "the second album must produce its own turn")
}

// A collection timer must not outlive the channel: stopping flushes whatever is
// pending rather than leaving it buffered forever.
func TestPendingAlbumIsFlushedOnStop(t *testing.T) {
	messageBus, ch := newMediaGroupTestChannel(time.Hour)

	msg := testMediaGroupMessage("album-pending")
	msg.MessageID = 1
	msg.Caption = "still collecting"
	require.NoError(t, ch.handleMessage(context.Background(), &msg))

	ch.flushPendingMediaGroups(context.Background())

	select {
	case inbound := <-messageBus.InboundChan():
		assert.Equal(t, "still collecting", inbound.Content)
	case <-time.After(time.Second):
		t.Fatal("a pending album was left buffered after shutdown")
	}

	ch.mediaGroupMu.Lock()
	_, stillPending := ch.mediaGroups["456:album-pending"]
	ch.mediaGroupMu.Unlock()
	assert.False(t, stillPending, "the collection timer must not leak past the flush")
}

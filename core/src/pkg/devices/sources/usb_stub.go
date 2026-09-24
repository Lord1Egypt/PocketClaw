//go:build !linux || android

// Android satisfies the linux build tag but has no udevadm — neither toybox nor
// the PocketClaw Managed Runtime ships it — and an app cannot subscribe to
// kernel uevents, so the udevadm-based monitor could only fail there. The
// Android Core compiles this inert monitor instead (PC-DEF-080).

package sources

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/devices/events"
)

type USBMonitor struct{}

func NewUSBMonitor() *USBMonitor {
	return &USBMonitor{}
}

func (m *USBMonitor) Kind() events.Kind {
	return events.KindUSB
}

func (m *USBMonitor) Start(ctx context.Context) (<-chan *events.DeviceEvent, error) {
	ch := make(chan *events.DeviceEvent)
	close(ch) // Immediately close, no events
	return ch, nil
}

func (m *USBMonitor) Stop() error {
	return nil
}

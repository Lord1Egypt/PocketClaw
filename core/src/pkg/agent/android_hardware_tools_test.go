package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// D (end to end). Registration is gated on IsToolEnabled, so a config that
// resolves the host-bus tools as disabled is what keeps them out of the tool
// registry and therefore out of the definitions the model is sent.
//
// PocketClaw Android forces that resolution through the environment its managed
// Gateway is launched with; this asserts the consequence, which is the part the
// model actually sees.
func TestDisabledHardwareToolsNeverReachTheModelDefinitions(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	if cfg.Tools.IsToolEnabled("i2c") || cfg.Tools.IsToolEnabled("spi") ||
		cfg.Tools.IsToolEnabled("serial") {
		t.Fatal("expected the host-bus tools to resolve as disabled here")
	}

	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	agent := loop.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected a default agent")
	}

	for _, name := range []string{"i2c", "spi", "serial"} {
		if _, ok := agent.Tools.Get(name); ok {
			t.Errorf("%s is registered in ToolRegistry despite being disabled", name)
		}
	}

	for _, def := range agent.Tools.ToProviderDefs() {
		switch def.Function.Name {
		case "i2c", "spi", "serial":
			t.Errorf("%s reached the model tool definitions despite being disabled",
				def.Function.Name)
		}
	}

	// The registry is otherwise populated, so the assertion above is not
	// passing merely because nothing was registered at all.
	if len(agent.Tools.List()) == 0 {
		t.Fatal("no tools registered; the assertions above would prove nothing")
	}
}

// The same wiring must still register them where they are enabled and usable,
// so this rule is a product boundary rather than a removal.
func TestEnabledHardwareToolsStillRegister(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.I2C.Enabled = true
	cfg.Tools.SPI.Enabled = true
	cfg.Tools.Serial.Enabled = true

	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	agent := loop.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected a default agent")
	}

	registered := map[string]bool{}
	for _, def := range agent.Tools.ToProviderDefs() {
		registered[def.Function.Name] = true
	}
	for _, name := range []string{"i2c", "spi", "serial"} {
		if _, ok := agent.Tools.Get(name); !ok {
			t.Errorf("%s did not register when enabled; the tools were removed rather "+
				"than gated", name)
		}
		if !registered[name] {
			t.Errorf("%s did not reach the model definitions when enabled", name)
		}
	}
}

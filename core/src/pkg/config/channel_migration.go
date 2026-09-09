package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// channelContainerKeys are the object keys a channel map has been stored under,
// current first. loadSecurityConfig accepts both for the same reason.
var channelContainerKeys = []string{"channel_list", "channels"}

// ErrChannelMigrationConflict is returned when an installation holds both a
// legacy and a canonical definition of the managed channel and they disagree.
var ErrChannelMigrationConflict = errors.New("conflicting channel configuration")

// migrateChannelIdentities rewrites a pre-migration config.json and its sibling
// .security.yml to the canonical channel names, once.
//
// This is a targeted edit of the serialized documents, not a load-and-save.
// SaveConfig marshals the typed Config, which is lossy for anything the struct
// does not model, and a namespace migration is the wrong moment to discover
// that: the user did not ask for their configuration to be rewritten, only for
// the channel to be renamed. Unrelated entries are carried across as raw bytes
// and keys keep their order, so the diff on the user's file is the rename and
// nothing else.
//
// Returns whether anything changed.
func migrateChannelIdentities(configPath string) (bool, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		// A missing or unreadable config is not this function's problem;
		// LoadConfig reports it with the diagnostics it already has.
		return false, nil
	}

	migrated, changed, err := migrateChannelIdentitiesJSON(raw)
	if err != nil {
		return false, err
	}

	securityFile := securityPath(configPath)
	securityRaw, securityErr := os.ReadFile(securityFile)
	var securityMigrated []byte
	securityChanged := false
	if securityErr == nil {
		securityMigrated, securityChanged, err = migrateChannelIdentitiesYAML(securityRaw)
		if err != nil {
			return false, err
		}
	}

	if !changed && !securityChanged {
		return false, nil
	}

	// The credential file goes first. .security.yml holds the channel token, so
	// a crash between the two writes must not leave config.json naming a
	// channel whose token is still filed under the old name — the token would
	// silently stop being found. The other order is recoverable: a legacy
	// config.json with a canonical .security.yml migrates again on next start.
	if securityChanged {
		if err := fileutil.WriteFileAtomic(securityFile, securityMigrated, 0o600); err != nil {
			return false, fmt.Errorf("failed to migrate channel identity in %s: %w", securityFile, err)
		}
	}
	if changed {
		if err := fileutil.WriteFileAtomic(configPath, migrated, 0o600); err != nil {
			return false, fmt.Errorf("failed to migrate channel identity in %s: %w", configPath, err)
		}
	}

	logger.InfoCF("config", "migrated the managed channel to its canonical name", map[string]any{
		"config":   changed,
		"security": securityChanged,
	})
	return true, nil
}

// migrateChannelIdentitiesJSON renames the legacy channel entries in a
// serialized config document.
func migrateChannelIdentitiesJSON(raw []byte) ([]byte, bool, error) {
	root, err := decodeOrderedObject(raw)
	if err != nil {
		// Not an object we can edit safely. LoadConfig's own diagnostics are
		// better at explaining a malformed config than anything here.
		return raw, false, nil
	}

	// "channel_list" is the current key; "channels" is what a config written
	// before the v3 format migration still uses. The identity rename is
	// orthogonal to that version migration and runs ahead of it, so both
	// containers have to be recognised.
	container := ""
	var channelsRaw json.RawMessage
	for _, candidate := range channelContainerKeys {
		if value, ok := root.get(candidate); ok {
			container, channelsRaw = candidate, value
			break
		}
	}
	if container == "" {
		return raw, false, nil
	}
	channels, err := decodeOrderedObject(channelsRaw)
	if err != nil {
		return raw, false, nil
	}

	changed := false
	for legacy, canonical := range legacyChannelNames {
		legacyEntry, hasLegacy := channels.get(legacy)
		if !hasLegacy {
			continue
		}
		canonicalEntry, hasCanonical := channels.get(canonical)
		if hasCanonical {
			// Both definitions exist. Identical ones are one channel written
			// twice and the legacy copy is redundant; different ones are two
			// different sets of settings, possibly two different tokens, and
			// nothing in the file says which the user meant.
			if !equivalentChannelEntries(legacyEntry, canonicalEntry) {
				return nil, false, fmt.Errorf(
					"%w: channels.%s and channels.%s both exist and differ; "+
						"neither was changed. Remove or reconcile one of them",
					ErrChannelMigrationConflict, legacy, canonical)
			}
			channels.remove(legacy)
			changed = true
			continue
		}

		renamed, err := renameChannelEntry(legacyEntry, legacy, canonical)
		if err != nil {
			return nil, false, err
		}
		channels.rename(legacy, canonical, renamed)
		changed = true
	}

	if !changed {
		return raw, false, nil
	}

	encoded, err := channels.encode(1)
	if err != nil {
		return nil, false, err
	}
	root.set(container, encoded)
	out, err := root.encode(0)
	if err != nil {
		return nil, false, err
	}
	return append(out, '\n'), true, nil
}

// renameChannelEntry updates the identity carried inside one channel entry: its
// "type", and the owner principal in its allow list.
func renameChannelEntry(entry json.RawMessage, legacy, canonical string) (json.RawMessage, error) {
	object, err := decodeOrderedObject(entry)
	if err != nil {
		return entry, nil
	}

	if typeRaw, ok := object.get("type"); ok {
		var value string
		if json.Unmarshal(typeRaw, &value) == nil && value == legacy {
			encoded, err := json.Marshal(canonical)
			if err != nil {
				return nil, err
			}
			object.set("type", encoded)
		}
	}

	if allowRaw, ok := object.get("allow_from"); ok {
		migrated, changed, err := migrateOwnerPrincipal(allowRaw, ownerPrincipalIndent)
		if err != nil {
			return nil, err
		}
		if changed {
			object.set("allow_from", migrated)
		}
	}

	return object.encode(2)
}

// migrateOwnerPrincipal rewrites the legacy owner label wherever it appears in
// a channel's allow list.
//
// Scoped to that field alone. The same string somewhere else in the user's
// configuration is a value we know nothing about, and a namespace migration has
// no business editing it.
func migrateOwnerPrincipal(raw json.RawMessage, indent string) (json.RawMessage, bool, error) {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		if single != LegacyPocketClawOwnerPrincipal {
			return raw, false, nil
		}
		encoded, err := json.Marshal(PocketClawOwnerPrincipal)
		return encoded, err == nil, err
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return raw, false, nil
	}

	changed := false
	seen := make(map[string]bool, len(list))
	out := make([]string, 0, len(list))
	for _, entry := range list {
		if entry == LegacyPocketClawOwnerPrincipal {
			entry = PocketClawOwnerPrincipal
			changed = true
		}
		// Both spellings in one list collapse to one owner. That is a rename,
		// not a widening: the resulting list is never longer than it was.
		if seen[entry] {
			changed = true
			continue
		}
		seen[entry] = true
		out = append(out, entry)
	}
	if !changed {
		return raw, false, nil
	}
	encoded, err := json.MarshalIndent(out, indent, "  ")
	return encoded, err == nil, err
}

// ownerPrincipalIndent is the leading indent of an allow_from value: three
// levels in, under channels > <channel>. It is what keeps the rewritten list
// aligned with the rest of the file.
const ownerPrincipalIndent = "      "

// equivalentChannelEntries reports whether two channel definitions say the same
// thing once their type fields are set aside.
//
// The type differs by definition here — that is what is being migrated — so it
// is normalized away and everything else must match exactly. Anything this
// cannot prove equal is treated as a conflict.
func equivalentChannelEntries(legacy, canonical json.RawMessage) bool {
	normalize := func(raw json.RawMessage) (string, bool) {
		var value map[string]any
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", false
		}
		delete(value, "type")
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", false
		}
		return string(encoded), true
	}
	left, okLeft := normalize(legacy)
	right, okRight := normalize(canonical)
	return okLeft && okRight && left == right
}

// migrateChannelIdentitiesYAML renames the legacy channel keys in .security.yml.
//
// The channel token lives here, filed under the channel name, and it is merged
// back by name at load. Renaming the channel in config.json without renaming it
// here would leave the token behind under a name nothing looks for any more.
//
// yaml.Node keeps ordering, comments and formatting, so only the key moves.
func migrateChannelIdentitiesYAML(raw []byte) ([]byte, bool, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return raw, false, nil
	}
	if len(document.Content) == 0 {
		return raw, false, nil
	}

	var channels *yaml.Node
	for _, candidate := range channelContainerKeys {
		if node := mappingValue(document.Content[0], candidate); node != nil {
			channels = node
			break
		}
	}
	if channels == nil {
		return raw, false, nil
	}

	changed := false
	for legacy, canonical := range legacyChannelNames {
		legacyKey := mappingKeyNode(channels, legacy)
		if legacyKey == nil {
			continue
		}
		if mappingValue(channels, canonical) != nil {
			// The JSON side decides conflicts; here the canonical entry already
			// exists and is authoritative, so the legacy key is dropped rather
			// than merged over it.
			removeMappingKey(channels, legacy)
			changed = true
			continue
		}
		legacyKey.Value = canonical
		changed = true
	}

	if !changed {
		return raw, false, nil
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, false, err
	}
	if err := encoder.Close(); err != nil {
		return nil, false, err
	}
	return buf.Bytes(), true, nil
}

func mappingKeyNode(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i]
		}
	}
	return nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func removeMappingKey(mapping *yaml.Node, key string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

// orderedObject is a JSON object that survives a decode/encode round trip with
// its key order and its untouched values byte-identical.
//
// encoding/json alone cannot do this: a map reorders keys and a struct drops
// what it does not model. Both would turn a one-line rename into a wholesale
// reformat of the user's configuration.
type orderedObject struct {
	keys   []string
	values map[string]json.RawMessage
}

func decodeOrderedObject(raw []byte) (*orderedObject, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("not a JSON object")
	}

	object := &orderedObject{values: make(map[string]json.RawMessage)}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("non-string object key")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		if _, seen := object.values[key]; !seen {
			object.keys = append(object.keys, key)
		}
		object.values[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return object, nil
}

func (o *orderedObject) get(key string) (json.RawMessage, bool) {
	value, ok := o.values[key]
	return value, ok
}

func (o *orderedObject) set(key string, value json.RawMessage) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *orderedObject) remove(key string) {
	if _, exists := o.values[key]; !exists {
		return
	}
	delete(o.values, key)
	for i, existing := range o.keys {
		if existing == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

// rename replaces one key in place, so the entry keeps its position in the file.
func (o *orderedObject) rename(from, to string, value json.RawMessage) {
	for i, existing := range o.keys {
		if existing == from {
			o.keys[i] = to
			delete(o.values, from)
			o.values[to] = value
			return
		}
	}
	o.set(to, value)
}

// encode writes the object indented for a value nested `depth` levels deep,
// matching what json.MarshalIndent(cfg, "", "  ") produced when the file was
// written. Untouched values are emitted exactly as they were read, so their
// own inner indentation is already correct for the position they keep.
func (o *orderedObject) encode(depth int) ([]byte, error) {
	if len(o.keys) == 0 {
		return []byte("{}"), nil
	}
	indent := strings.Repeat("  ", depth+1)
	var buf bytes.Buffer
	buf.WriteString("{\n")
	for i, key := range o.keys {
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.WriteString(indent)
		buf.Write(encodedKey)
		buf.WriteString(": ")
		buf.Write(o.values[key])
		if i < len(o.keys)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString(strings.Repeat("  ", depth))
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

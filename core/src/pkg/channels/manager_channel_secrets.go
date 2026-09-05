package channels

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// secretFingerprintPrefix marks a value in the reconcile map as a digest of a
// credential rather than the credential itself.
const secretFingerprintPrefix = "sha256:"

var (
	secureStringType  = reflect.TypeOf(config.SecureString{})
	secureStringsType = reflect.TypeOf(config.SecureStrings{})
)

// injectSecretFingerprints rewrites every credential field of a decoded channel
// settings struct into the JSON-shaped map used to compute the reconcile hash.
//
// config.SecureString marshals to a fixed placeholder, so a channel's token is
// invisible to json.Marshal. Without this the reconcile hash cannot see a token
// change at all, and saving a new bot token never restarts the channel — the
// user has to stop and start PocketClaw by hand.
//
// The digest, not the secret, is what lands in the map: change detection needs
// only to know that the value differs, and the resulting hash is kept in memory
// for the lifetime of the manager.
func injectSecretFingerprints(target map[string]any, decoded any) {
	if target == nil || decoded == nil {
		return
	}
	injectSecretFingerprintsValue(target, reflect.ValueOf(decoded))
}

func injectSecretFingerprintsValue(target map[string]any, value reflect.Value) {
	value = derefValue(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return
	}

	structType := value.Type()
	for i := range structType.NumField() {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		name := jsonFieldName(field)
		if name == "" {
			continue
		}
		injectFieldFingerprint(target, name, value.Field(i))
	}
}

func injectFieldFingerprint(target map[string]any, name string, fieldValue reflect.Value) {
	switch {
	case fieldValue.Type() == secureStringType:
		secure, ok := fieldValue.Interface().(config.SecureString)
		if !ok {
			return
		}
		target[name] = fingerprintSecret(secure.String())
	case fieldValue.Type() == secureStringsType:
		secures, ok := fieldValue.Interface().(config.SecureStrings)
		if !ok {
			return
		}
		digests := make([]string, 0, len(secures))
		for _, secure := range secures.Values() {
			digests = append(digests, fingerprintSecret(secure))
		}
		target[name] = digests
	default:
		injectNestedFingerprints(target, name, fieldValue)
	}
}

// injectNestedFingerprints recurses into the containers a channel uses to hold
// credentials: a nested settings struct, or a map of named targets such as the
// Slack and Teams webhook tables.
func injectNestedFingerprints(target map[string]any, name string, fieldValue reflect.Value) {
	fieldValue = derefValue(fieldValue)
	if !fieldValue.IsValid() {
		return
	}

	switch fieldValue.Kind() {
	case reflect.Struct:
		if !structContainsSecret(fieldValue.Type()) {
			return
		}
		nested := childMap(target, name)
		injectSecretFingerprintsValue(nested, fieldValue)
	case reflect.Map:
		if fieldValue.Type().Key().Kind() != reflect.String ||
			!structContainsSecret(fieldValue.Type().Elem()) {
			return
		}
		nested := childMap(target, name)
		for _, key := range fieldValue.MapKeys() {
			entry := childMap(nested, key.String())
			injectSecretFingerprintsValue(entry, fieldValue.MapIndex(key))
		}
		// Entries removed from the map must not survive in the hash input.
		for existing := range nested {
			if !fieldValue.MapIndex(reflect.ValueOf(existing)).IsValid() {
				delete(nested, existing)
			}
		}
	}
}

// childMap returns the nested map stored under name, replacing whatever the
// JSON marshaller left there when it is not a map (a nil, or a placeholder).
func childMap(target map[string]any, name string) map[string]any {
	if existing, ok := target[name].(map[string]any); ok && existing != nil {
		return existing
	}
	created := make(map[string]any)
	target[name] = created
	return created
}

func structContainsSecret(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch {
	case t == secureStringType || t == secureStringsType:
		return true
	case t.Kind() == reflect.Map:
		return structContainsSecret(t.Elem())
	case t.Kind() != reflect.Struct:
		return false
	}
	for i := range t.NumField() {
		if !t.Field(i).IsExported() {
			continue
		}
		if structContainsSecret(t.Field(i).Type) {
			return true
		}
	}
	return false
}

func derefValue(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return field.Name
}

// fingerprintSecret digests a credential for change detection. An empty
// credential keeps an empty marker so configuring a token for the first time is
// still seen as a change.
func fingerprintSecret(secret string) string {
	if secret == "" {
		return secretFingerprintPrefix + "empty"
	}
	sum := sha256.Sum256([]byte(secret))
	return secretFingerprintPrefix + hex.EncodeToString(sum[:])
}

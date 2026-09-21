package api

import (
	"sort"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// computeModelCredentialSignatures returns one entry per model_list index
// describing the connection material the gateway resolves when it boots: which
// provider and model it talks to, at what endpoint, through what proxy, with
// what credential, and with what extra headers.
//
// It exists because the rest of computeConfigSignature covers only the default
// model name and the streaming flag of referenced entries. Rotating an API key
// therefore left the signature unchanged, so gateway_restart_required stayed
// false, the console reported the save as applied, and the running gateway went
// on using the previous credential until something unrelated forced a restart.
//
// Every entry is covered rather than only the ones reachable from the default
// and fallback chains. A key belonging to any configured model is material the
// booted process holds, and narrowing this to referenced entries would restore
// the same class of silent staleness for the rest.
//
// Secrets are reduced to a digest. This value is only ever compared against the
// signature the gateway booted with and is never serialised to a client, but it
// is the kind of value that ends up in a debug log, and a comparison does not
// need the plaintext.
func computeModelCredentialSignatures(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	signatures := make([]string, 0, len(cfg.ModelList))
	for i, mc := range cfg.ModelList {
		if mc == nil {
			continue
		}
		signatures = append(signatures, strings.Join([]string{
			strconv.Itoa(i),
			strings.TrimSpace(mc.ModelName),
			strings.TrimSpace(mc.Provider),
			strings.TrimSpace(mc.Model),
			strings.TrimSpace(mc.APIBase),
			strings.TrimSpace(mc.Proxy),
			strings.TrimSpace(mc.AuthMethod),
			strconv.FormatBool(mc.Enabled),
			apiKeysDigest(mc),
			customHeadersDigest(mc.CustomHeaders),
		}, "|"))
	}
	sort.Strings(signatures)
	return signatures
}

// apiKeysDigest reduces a model's whole key list to one short digest.
//
// The list, not just the first key, because multi-key entries fail over between
// them: replacing a second key is as much a runtime change as replacing the
// first. Position is part of the digest for the same reason — reordering keys
// changes which one is tried first.
func apiKeysDigest(mc *config.ModelConfig) string {
	if mc == nil || len(mc.APIKeys) == 0 {
		return "nokey"
	}
	parts := make([]string, 0, len(mc.APIKeys)*2)
	for i, key := range mc.APIKeys {
		parts = append(parts, strconv.Itoa(i), key.String())
	}
	return signatureDigest(parts...)
}

// customHeadersDigest reduces per-model headers to one digest over a sorted
// walk, so an unchanged map always produces the same value regardless of Go's
// map iteration order. Header values can carry credentials, which is the other
// reason this is a digest rather than the map itself.
func customHeadersDigest(headers map[string]string) string {
	if len(headers) == 0 {
		return "noheaders"
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names)*2)
	for _, name := range names {
		parts = append(parts, name, headers[name])
	}
	return signatureDigest(parts...)
}

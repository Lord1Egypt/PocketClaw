// Package bootstrap records what PocketClaw seeded into a workspace, so a later
// version can tell what it is allowed to touch.
//
// Seeding is deliberately one-directional: a bundled template is written only
// when the file is absent, and after that the file belongs to the user. That
// keeps edits safe and creates the opposite problem — an install seeded before a
// default improved keeps the old text forever, with nothing on disk recording
// which version it came from.
//
// This package writes that record. It is small on purpose: for each seeded
// bootstrap document it stores the digest of exactly what PocketClaw wrote, so a
// later upgrade can distinguish three states without diffing prose or keeping
// every historical template around:
//
//   - recorded, digest still matches → PocketClaw's own text, untouched
//   - recorded, digest differs       → the user edited it; theirs wins, always
//   - not recorded                   → provenance unknown; never assume
//
// Nothing here modifies a workspace file, and memory is recorded but never
// inspected: MEMORY.md is the user's, and a bootstrap step has no business
// reading it.
//
// # This record is advisory, not an authorization boundary
//
// The file lives in the workspace, which the user can read, edit and delete,
// and which on Android may sit on shared storage. It is therefore **untrusted
// input**. It may describe what PocketClaw previously seeded; it may never, on
// its own, authorize overwriting, replacing or deleting a user-owned file.
//
// Legitimate uses: deciding a non-destructive migration, offering an upgrade,
// noticing a probably-untouched default, diagnostics.
//
// Not a legitimate use: treating ProvenanceMatchesSeed as permission to write.
// Anyone who can edit this file can make any document look pristine simply by
// recording the digest of its current contents. That is not an escalation —
// they could edit the document directly — but it must not become a way to make
// *PocketClaw* destroy the document on their behalf. Nothing in this package
// returns "you may overwrite this", and nothing should be added that does: a
// destructive change needs a decision from the user at the time, which a file
// on disk cannot stand in for.
//
// Every ambiguity resolves the same way. Missing record, unreadable file,
// malformed JSON, a schema version this build does not know, no entry for the
// document, an entry whose digest does not match, an unreadable document — all
// of them mean ProvenanceUnknown or ProvenanceUserModified, and both mean leave
// it alone.
package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Version is the schema version of the metadata file. Bump it only for a change
// a reader must notice; new optional fields do not need it.
const Version = 1

// MetadataDir and MetadataFile name the on-disk record, relative to the
// workspace root.
//
// COMPATIBILITY: this is new state introduced with the PocketClaw name, so it
// is deliberately not "picoclaw". It needs no rename, which is the point.
// RECHECK AFTER FULL NAMESPACE MIGRATION.
const (
	MetadataDir  = ".pocketclaw"
	MetadataFile = "bootstrap.json"
)

// Owner describes who a seeded file belongs to after it is written.
type Owner string

const (
	// OwnerUser marks a document seeded once and owned by the user from then
	// on. PocketClaw may read it, and may never rewrite it.
	OwnerUser Owner = "user"
	// OwnerUserPrivate marks the user's memory. Recorded so the seed is
	// accounted for, and never read, diffed or migrated by a bootstrap step.
	OwnerUserPrivate Owner = "user-private"
)

// TemplateRecord is what PocketClaw wrote for one bootstrap document.
type TemplateRecord struct {
	Owner Owner `json:"owner"`
	// SeededSHA256 is the digest of the bytes PocketClaw wrote, not of the
	// file as it stands now. That is what makes "has the user touched this?"
	// answerable later.
	SeededSHA256 string `json:"seededSha256"`
	SeededAt     string `json:"seededAt"`
}

// Metadata is the whole record.
type Metadata struct {
	BootstrapVersion int `json:"bootstrapVersion"`
	// ManagedGuidanceVersion is the revision of the guidance the binary
	// supplied at seed time. Guidance itself ships in the binary, so this is
	// provenance rather than content.
	ManagedGuidanceVersion int                       `json:"managedGuidanceVersion"`
	Templates              map[string]TemplateRecord `json:"templates"`
}

// TrackedTemplates are the bootstrap documents whose ownership this record is
// about, mapped to their owner. Everything else PocketClaw seeds — skills and
// their assets — is replaceable product content and is deliberately not tracked
// here.
//
// COMPATIBILITY: these are workspace-relative file names, unchanged by the
// namespace migration, but listed here so the migration audit can see them.
// RECHECK AFTER FULL NAMESPACE MIGRATION.
var TrackedTemplates = map[string]Owner{
	"AGENT.md":         OwnerUser,
	"SOUL.md":          OwnerUser,
	"USER.md":          OwnerUser,
	"memory/MEMORY.md": OwnerUserPrivate,
}

// Path returns the metadata file's location for a workspace.
func Path(workspace string) string {
	return filepath.Join(workspace, MetadataDir, MetadataFile)
}

// Load reads the record for a workspace. A workspace that has none — every
// install seeded before this existed — returns a zero Metadata and no error,
// because "no record" is a normal state that means "assume nothing".
func Load(workspace string) (Metadata, error) {
	data, err := os.ReadFile(Path(workspace))
	if errors.Is(err, os.ErrNotExist) {
		return Metadata{}, nil
	}
	if err != nil {
		return Metadata{}, fmt.Errorf("read bootstrap metadata: %w", err)
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, fmt.Errorf("parse bootstrap metadata: %w", err)
	}
	return meta, nil
}

// Record adds entries for the files a seeding run actually wrote and saves the
// result.
//
// Existing entries are never replaced. A rerun writes nothing new, so it records
// nothing new, and a file the user has since edited keeps the digest of what was
// originally seeded — overwriting it with the current contents would relabel the
// user's own text as PocketClaw's.
//
// seeded maps workspace-relative paths to the bytes that were written. Paths
// outside TrackedTemplates are ignored.
func Record(workspace string, guidanceVersion int, seeded map[string][]byte) error {
	meta, err := Load(workspace)
	if err != nil {
		return err
	}

	meta.BootstrapVersion = Version
	if meta.ManagedGuidanceVersion == 0 {
		meta.ManagedGuidanceVersion = guidanceVersion
	}
	if meta.Templates == nil {
		meta.Templates = map[string]TemplateRecord{}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	added := false
	for _, rel := range sortedKeys(seeded) {
		owner, tracked := TrackedTemplates[rel]
		if !tracked {
			continue
		}
		if _, exists := meta.Templates[rel]; exists {
			continue
		}
		sum := sha256.Sum256(seeded[rel])
		meta.Templates[rel] = TemplateRecord{
			Owner:        owner,
			SeededSHA256: hex.EncodeToString(sum[:]),
			SeededAt:     now,
		}
		added = true
	}

	// A rerun that seeded nothing new must leave the file byte-identical, so
	// there is nothing to prove about repeated writes.
	if !added && meta.BootstrapVersion == Version {
		if _, err := os.Stat(Path(workspace)); err == nil {
			return nil
		}
	}

	return save(workspace, meta)
}

// ProvenanceState is what the record can say about one document. It is a
// description, never a permission — see the package comment.
type ProvenanceState string

const (
	// ProvenanceUnknown means nothing here can be relied on: no record, an
	// unreadable or malformed one, a schema version this build does not
	// understand, no entry for this document, or a document that cannot be
	// read. Treat the document as the user's.
	ProvenanceUnknown ProvenanceState = "unknown"
	// ProvenanceUserModified means the document differs from what PocketClaw
	// seeded. It is theirs.
	ProvenanceUserModified ProvenanceState = "user-modified"
	// ProvenanceMatchesSeed means the document is byte-identical to what
	// PocketClaw recorded seeding.
	//
	// This is a hint for offering an upgrade, and it is not permission to
	// perform one. The record is user-writable, so this state can be
	// manufactured; a destructive change needs the user's decision as well.
	ProvenanceMatchesSeed ProvenanceState = "matches-seed"
)

// Provenance answers what is known about one tracked document.
//
// It never reports ProvenanceMatchesSeed for memory: MEMORY.md is recorded so
// the seed is accounted for and is deliberately never read here, so it cannot
// become an upgrade candidate even if something later forgets that it must not.
func (m Metadata) Provenance(workspace, rel string) ProvenanceState {
	// A schema this build does not understand may mean the fields do not mean
	// what they appear to. Refuse to interpret it.
	if m.BootstrapVersion <= 0 || m.BootstrapVersion > Version {
		return ProvenanceUnknown
	}
	if _, tracked := TrackedTemplates[rel]; !tracked {
		return ProvenanceUnknown
	}
	record, recorded := m.Templates[rel]
	if !recorded || record.SeededSHA256 == "" {
		return ProvenanceUnknown
	}
	if record.Owner == OwnerUserPrivate {
		return ProvenanceUnknown
	}
	data, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(rel)))
	if err != nil {
		return ProvenanceUnknown
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != record.SeededSHA256 {
		return ProvenanceUserModified
	}
	return ProvenanceMatchesSeed
}

// UserOwns reports whether a tracked template must not be touched by an
// automatic upgrade. It is the safe reading of Provenance: everything that is
// not positively "this is the text we wrote, untouched" counts as the user's.
//
// Being false is still not permission to write — see the package comment.
func (m Metadata) UserOwns(workspace, rel string) bool {
	return m.Provenance(workspace, rel) != ProvenanceMatchesSeed
}

func save(workspace string, meta Metadata) error {
	dir := filepath.Join(workspace, MetadataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create bootstrap metadata directory: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode bootstrap metadata: %w", err)
	}
	data = append(data, '\n')

	// Temp-and-rename within the same directory: a workspace can be on shared
	// storage, and a half-written record is worse than none.
	tmp, err := os.CreateTemp(dir, MetadataFile+".*")
	if err != nil {
		return fmt.Errorf("create bootstrap metadata temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write bootstrap metadata: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync bootstrap metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close bootstrap metadata: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod bootstrap metadata: %w", err)
	}
	if err := os.Rename(tmpName, Path(workspace)); err != nil {
		return fmt.Errorf("install bootstrap metadata: %w", err)
	}
	return nil
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Package pcruntime is the PocketClaw Managed Runtime.
//
// It gives the agent a controlled, observable, verified local tool environment.
// The agent names a tool ("sha256sum", "jq"); the runtime resolves that name to
// a concrete executable through a versioned catalog, and runs it under bounded
// execution with a full structured lifecycle.
//
// # Why there is no provisioning code
//
// PocketClaw targets Android SDK 36. Since API 29 an app may not execve() a
// file in its own writable data directory, and File.setExecutable(true) does
// not change that: the restriction is enforced on the app's SELinux domain, not
// by the file mode. Executable payloads can therefore only reach the device by
// two routes, both read-only to the app:
//
//   - DeliverySystem: the platform's own /system/bin, shipped by the OS.
//   - DeliveryBundled: lib*.so entries in the APK, unpacked by the package
//     manager into nativeLibraryDir at install time.
//
// Downloading a binary and running it is not a policy this package refuses; it
// is a thing the platform cannot do. That is why no download path exists here
// to audit. EnsureTool reports capability-unavailable for anything outside the
// catalog rather than acquiring it.
package pcruntime

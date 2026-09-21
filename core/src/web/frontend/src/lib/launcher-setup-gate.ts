import type { LauncherAuthStatus } from "@/api/launcher-auth"

/**
 * What /launcher-setup should do for a given auth status.
 *
 * PC-DEF-037. The route used to render the first-run "create a dashboard
 * password" form to anyone who navigated to it, including an unauthenticated
 * visitor on an already-initialized dashboard. The backend always refused the
 * POST, so nothing could be taken over -- but the page invited the attempt and
 * announced to a stranger that a PocketClaw dashboard lives here.
 *
 * This is UX only. Authorization stays in the backend: first-claim setup is
 * loopback-only and an initialized dashboard requires a session (PC-DEF-039).
 */
export type LauncherSetupGate =
  | { render: "first-run" }
  | { render: "leaving"; redirectTo: string }

/**
 * `null` means the status call failed. Showing the first-run form then would be
 * a guess about who owns this dashboard, so the visitor goes to login: correct
 * whenever a password exists, and harmless when one does not.
 */
export function resolveLauncherSetupGate(
  status: LauncherAuthStatus | null,
): LauncherSetupGate {
  if (status === null) return { render: "leaving", redirectTo: "/launcher-login" }
  if (!status.initialized) return { render: "first-run" }

  // An owner already exists, so this is never first-run. An authenticated owner
  // goes to the existing password-change surface in Config rather than a
  // second, conflicting flow; anyone else logs in first.
  return {
    render: "leaving",
    redirectTo: status.authenticated ? "/config" : "/launcher-login",
  }
}

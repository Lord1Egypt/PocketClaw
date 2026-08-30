/**
 * Marks the console as mounted and rendering.
 *
 * The Android host cannot observe a dead WebView renderer: Android may kill a
 * backgrounded renderer process to reclaim memory, and
 * `webview_flutter_android` exposes no `onRenderProcessGone` callback. The host
 * therefore asks the page a question only a live page can answer, and this flag
 * is that answer.
 *
 * It is set once the router has actually rendered, not merely when the bundle
 * has parsed, so a page that loaded its script but failed to render still reads
 * as not ready.
 */
const READY_FLAG = "__pocketclawReady"

declare global {
  interface Window {
    __pocketclawReady?: boolean
  }
}

export function markAppReady() {
  try {
    window[READY_FLAG] = true
  } catch {
    // A page that cannot write to its own window is not one the host can
    // recover by asking it anything; the probe falls back to "not ready".
  }
}

/**
 * Clears the flag when the app can no longer be trusted to be rendering.
 *
 * Called from the error boundary: a crashed React tree leaves a shell on screen
 * that is not a usable console, and the host should be able to tell.
 */
export function markAppUnhealthy() {
  try {
    window[READY_FLAG] = false
  } catch {
    // Nothing to do; the probe already treats an unreadable page as unusable.
  }
}

export function isAppReady(): boolean {
  try {
    return window[READY_FLAG] === true
  } catch {
    return false
  }
}

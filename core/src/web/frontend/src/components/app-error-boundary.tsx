import { Component, type ErrorInfo, type ReactNode } from "react"

import { markAppUnhealthy } from "@/lib/app-readiness"

interface Props {
  children: ReactNode
}

interface State {
  failed: boolean
}

/**
 * Keeps a render error from emptying the page.
 *
 * Without a boundary, an uncaught error unmounts the whole React tree and
 * leaves `#root` empty — visually identical to a WebView whose renderer was
 * killed, and equally unexplained. This turns that into a visible message the
 * user can act on, and marks the app unhealthy so the Android host's resume
 * probe can recover it too.
 */
export class AppErrorBoundary extends Component<Props, State> {
  state: State = { failed: false }

  static getDerivedStateFromError(): State {
    return { failed: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    markAppUnhealthy()
    // Console only: an error message can quote request state, so it is not
    // sent anywhere it would be persisted.
    console.error("PocketClaw console crashed", error, info.componentStack)
  }

  private reload = () => {
    window.location.reload()
  }

  render() {
    if (!this.state.failed) return this.props.children

    return (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: "0.75rem",
          minHeight: "100vh",
          padding: "1.5rem",
          textAlign: "center",
          fontFamily: "system-ui, sans-serif",
        }}
      >
        <p style={{ margin: 0, fontSize: "0.95rem" }}>
          The console stopped responding.
        </p>
        <button
          type="button"
          onClick={this.reload}
          style={{
            padding: "0.5rem 1rem",
            borderRadius: "0.5rem",
            border: "1px solid currentColor",
            background: "transparent",
            cursor: "pointer",
            font: "inherit",
          }}
        >
          Reload
        </button>
      </div>
    )
  }
}

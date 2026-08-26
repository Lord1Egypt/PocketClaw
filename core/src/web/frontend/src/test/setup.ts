import { cleanup } from "@testing-library/react"
import { afterEach } from "vitest"

// jsdom under Node 25 can start without a usable Storage, and modules that
// build persisted atoms at import time crash before any test runs.
function installStorage(name: "localStorage" | "sessionStorage") {
  const existing = window[name] as Storage | undefined
  if (existing && typeof existing.getItem === "function") return
  const store = new Map<string, string>()
  Object.defineProperty(window, name, {
    configurable: true,
    value: {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => void store.set(key, String(value)),
      removeItem: (key: string) => void store.delete(key),
      clear: () => store.clear(),
      key: (index: number) => Array.from(store.keys())[index] ?? null,
      get length() {
        return store.size
      },
    } satisfies Storage,
  })
}

installStorage("localStorage")
installStorage("sessionStorage")

afterEach(() => {
  cleanup()
  delete window.__pocketclawHost
})

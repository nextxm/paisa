/**
 * Storage helpers shared by the persisted state manager.
 *
 * These are extracted into a plain `.ts` module so they can be unit-tested
 * without requiring the Svelte compiler (`.svelte.ts` cannot be imported
 * directly by the Bun test runner).
 */

/**
 * Read a JSON-serialised value from `localStorage`.
 * Returns `defaultValue` when the key is absent, `localStorage` is unavailable
 * (e.g. SSR / worker contexts), or the stored value cannot be parsed.
 */
export function loadFromStorage<T>(key: string, defaultValue: T): T {
  if (typeof localStorage === "undefined") return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (stored !== null) return JSON.parse(stored) as T;
  } catch {
    // ignore parse / access errors
  }
  return defaultValue;
}

/**
 * Write a value to `localStorage` as JSON.
 * Silently ignores errors (e.g. private-browsing quota, unavailable storage).
 */
export function saveToStorage<T>(key: string, value: T): void {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // ignore write errors
  }
}

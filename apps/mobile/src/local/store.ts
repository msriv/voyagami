/**
 * The on-device store. Screens and repositories only know this interface, so the SQLite
 * implementation can be swapped (PowerSync, P3) without touching them. See docs/sync.md.
 */
export interface LocalStore {
  get(key: string): Promise<unknown>;
  put(key: string, value: unknown): Promise<void>;
  remove(key: string): Promise<void>;
  clear(): Promise<void>;
  /** Bytes of JSON held, for the storage line in Profile. */
  sizeBytes(): Promise<number>;
}

/** Used by tests and as the fallback when SQLite cannot open. */
export function memoryStore(): LocalStore {
  const m = new Map<string, string>();
  return {
    get: async (k) => (m.has(k) ? JSON.parse(m.get(k) as string) : undefined),
    put: async (k, v) => void m.set(k, JSON.stringify(v)),
    remove: async (k) => void m.delete(k),
    clear: async () => m.clear(),
    sizeBytes: async () => [...m.values()].reduce((n, s) => n + s.length, 0),
  };
}

/** Bump when the table layout changes; see "Migrations" in docs/sync.md. */
const SCHEMA_VERSION = 1;

/** SQLite-backed store: one table of JSON documents keyed by a string. */
export function sqliteStore(name = "voyagami.db"): LocalStore {
  type Db = import("expo-sqlite").SQLiteDatabase;
  let opening: Promise<Db> | undefined;
  const db = () =>
    (opening ??= (async () => {
      const { openDatabaseAsync } = await import("expo-sqlite");
      const d = await openDatabaseAsync(name);
      await d.execAsync(
        "CREATE TABLE IF NOT EXISTS docs (key TEXT PRIMARY KEY NOT NULL, json TEXT NOT NULL, updated_at INTEGER NOT NULL);" +
          `PRAGMA user_version = ${SCHEMA_VERSION};`,
      );
      return d;
    })());
  return {
    async get(key) {
      const row = await (
        await db()
      ).getFirstAsync<{ json: string }>("SELECT json FROM docs WHERE key = ?", key);
      return row ? JSON.parse(row.json) : undefined;
    },
    async put(key, value) {
      await (
        await db()
      ).runAsync(
        "INSERT OR REPLACE INTO docs (key, json, updated_at) VALUES (?, ?, ?)",
        key,
        JSON.stringify(value),
        Date.now(),
      );
    },
    async remove(key) {
      await (await db()).runAsync("DELETE FROM docs WHERE key = ?", key);
    },
    async clear() {
      await (await db()).runAsync("DELETE FROM docs");
    },
    async sizeBytes() {
      const row = await (
        await db()
      ).getFirstAsync<{ n: number | null }>("SELECT SUM(LENGTH(json)) AS n FROM docs");
      return row?.n ?? 0;
    },
  };
}

let current: LocalStore | undefined;

/** The store the app uses. Opens SQLite on first use. */
export function localStore(): LocalStore {
  return (current ??= sqliteStore());
}

/** For tests, and to swap the implementation. */
export function setLocalStore(s: LocalStore | undefined) {
  current = s;
}

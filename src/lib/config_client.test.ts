import { afterAll, beforeEach, describe, expect, mock, test } from "bun:test";

import { derived, get, writable } from "svelte/store";

const loadingSet = mock((_value: boolean) => {});
const jobsStore = writable<Record<string, any>>({});
const jobsMock = {
  ...jobsStore,
  upsert: (job: { id: string }) => jobsStore.update((current) => ({ ...current, [job.id]: job })),
  updateById: (id: string, partial: Record<string, any>) => {
    let found = false;
    jobsStore.update((current) => {
      const existing = current[id];
      if (!existing) return current;
      found = true;
      return { ...current, [id]: { ...existing, ...partial } };
    });
    return found;
  },
  reset: () => jobsStore.set({}),
  snapshot: () => get(jobsStore)
};
const jobsListMock = derived(jobsStore, ($jobs) =>
  Object.values($jobs).sort(
    (a, b) =>
      new Date((a as any).created_at || 0).getTime() -
      new Date((b as any).created_at || 0).getTime()
  )
);
const isJobRunningMock = derived(jobsStore, ($jobs) =>
  Object.values($jobs).some((job: any) => job.status === "pending" || job.status === "running")
);
const runningJobMock = derived(
  jobsStore,
  ($jobs) =>
    Object.values($jobs).find((job: any) => job.status === "pending" || job.status === "running") ??
    null
);

mock.module("../store", () => ({
  loading: { set: loadingSet },
  accountTfIdf: writable(null),
  theme: writable("light"),
  month: writable("2024-01"),
  year: writable(""),
  dateMin: writable(null),
  dateMax: writable(null),
  editorState: writable({}),
  sheetEditorState: writable({}),
  willClearTippy: writable(0),
  willRefresh: writable(0),
  commandPaletteOpen: writable(false),
  refresh: async () => true,
  jobs: jobsMock,
  jobsList: jobsListMock,
  isJobRunning: isJobRunningMock,
  runningJob: runningJobMock
}));

let getConfigImpl: (request?: unknown) => Promise<any> = mock(async (_request?: unknown) => ({}));
let updateConfigImpl: (request?: unknown) => Promise<any> = mock(async (_request?: unknown) => ({
  success: true
}));
mock.module("$lib/connect_client", () => ({
  paisaClient: {
    getConfig: (request: unknown) => getConfigImpl(request),
    updateConfig: (request: unknown) => updateConfigImpl(request)
  }
}));

const { fetchConfig, updateConfig } = await import("./config_client");

describe("config_client", () => {
  beforeEach(() => {
    loadingSet.mockClear();
    getConfigImpl = mock(async (_request?: unknown) => ({}));
    updateConfigImpl = mock(async (_request?: unknown) => ({ success: true }));
  });

  test("fetchConfig maps connect response fields to existing config shape", async () => {
    getConfigImpl = mock(async () => ({
      config: { default_currency: "INR" },
      schema: { title: "Config" },
      accounts: ["Assets:Cash"],
      lastPriceUpdate: "2026-05-18T00:00:00Z",
      isJournalDirty: true,
      now: "2026-05-18T00:00:00Z"
    }));

    const result = await fetchConfig();

    expect(result.config.default_currency).toBe("INR");
    expect(result.schema.title).toBe("Config");
    expect(result.accounts).toEqual(["Assets:Cash"]);
    expect(result.last_price_update).toBe("2026-05-18T00:00:00Z");
    expect(result.is_journal_dirty).toBe(true);
    expect(result.now?.format("YYYY-MM-DD")).toBe("2026-05-18");
    expect(loadingSet).toHaveBeenCalledTimes(2);
    expect(loadingSet.mock.calls[0][0]).toBe(true);
    expect(loadingSet.mock.calls[1][0]).toBe(false);
  });

  test("fetchConfig respects background flag and does not toggle loading", async () => {
    getConfigImpl = mock(async () => ({
      config: {},
      schema: {},
      accounts: [],
      lastPriceUpdate: "",
      isJournalDirty: false
    }));

    await fetchConfig({ background: true });
    expect(loadingSet).not.toHaveBeenCalled();
  });

  test("updateConfig returns structured error on connect failure", async () => {
    updateConfigImpl = mock(async () => {
      throw new Error("invalid config");
    });

    const result = await updateConfig({ default_currency: "INR" } as UserConfig);
    expect(result).toEqual({ success: false, error: "invalid config" });
  });

  afterAll(() => {
    mock.restore();
  });
});

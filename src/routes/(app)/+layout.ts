export const trailingSlash = "never";

import type { LayoutLoad } from "./$types";
import { configUpdated, setNow } from "$lib/utils";
import { fetchConfig } from "$lib/config_client";

export const load = (async () => {
  const { config, now, last_price_update, is_journal_dirty } = await fetchConfig();
  if (now) {
    setNow(now);
  }
  globalThis.USER_CONFIG = config;
  globalThis.USER_CONFIG.last_price_update = last_price_update;
  globalThis.USER_CONFIG.is_journal_dirty = is_journal_dirty;
  configUpdated();
  return {};
}) satisfies LayoutLoad;

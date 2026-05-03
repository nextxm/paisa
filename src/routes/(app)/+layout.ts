export const trailingSlash = "never";

import type { LayoutLoad } from "./$types";
import { ajax, configUpdated, setNow } from "$lib/utils";

export const load = (async () => {
  const { config, now, last_price_update, is_journal_dirty } = await ajax("/api/config");
  if (now) {
    setNow(now);
  }
  globalThis.USER_CONFIG = config;
  globalThis.USER_CONFIG.last_price_update = last_price_update;
  globalThis.USER_CONFIG.is_journal_dirty = is_journal_dirty;
  configUpdated();
  return {};
}) satisfies LayoutLoad;

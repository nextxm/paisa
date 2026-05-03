export const trailingSlash = "never";

import type { LayoutLoad } from "./$types";
import { ajax, configUpdated, setNow } from "$lib/utils";

export const load = (async () => {
  const { config, now, last_price_update } = await ajax("/api/config");
  if (now) {
    setNow(now);
  }
  globalThis.USER_CONFIG = config;
  globalThis.USER_CONFIG.last_price_update = last_price_update;
  configUpdated();
  return {};
}) satisfies LayoutLoad;

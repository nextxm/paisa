import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [networth, currencies] = await Promise.all([
    ajax("/api/networth"),
    ajax("/api/price/currencies")
  ]);

  return {
    networth,
    currencies
  };
}) satisfies PageLoad;

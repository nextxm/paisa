import type { PageLoad } from "./$types";

export const load = (async ({ params }) => {
  return {
    account: decodeURIComponent(params.name)
  };
}) satisfies PageLoad;

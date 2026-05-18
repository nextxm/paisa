import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async ({ params }) => {
  const goal = await ajax("/api/goals/savings/:name", undefined, { name: params.slug });

  return {
    name: params.slug,
    goal
  };
}) satisfies PageLoad;

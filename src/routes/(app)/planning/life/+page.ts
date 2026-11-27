import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const prerender = false;

export const load = (() => {
  throw redirect(301, "/planning/projection?mode=montecarlo");
}) satisfies PageLoad;

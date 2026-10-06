import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load = (() => {
  throw redirect(301, "/planning/tax/capital_gains");
}) satisfies PageLoad;

import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load = (({ params }) => {
  throw redirect(301, `/planning/goals/retirement/${params.slug}`);
}) satisfies PageLoad;

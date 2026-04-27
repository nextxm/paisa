/**
 * connect_client.ts — typed Connect-RPC transport client.
 *
 * Usage:
 *   import { paisaClient } from "$lib/connect_client";
 *   const { accounts } = await paisaClient.getAccountTree({});
 *
 * The client reuses the same auth token stored in localStorage by the
 * existing `ajax` helper so auth behaviour is identical on both paths.
 *
 * To regenerate `$lib/gen/api_pb.ts` after changing proto/api.proto:
 *
 *   PATH="$PWD/node_modules/.bin:$PATH" protoc \
 *     --proto_path=proto \
 *     --es_out=src/lib/gen \
 *     --es_opt=target=ts \
 *     proto/api.proto
 *
 * (Alternatively run `npm run generate:proto`.)
 */

import { createClient, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { PaisaService } from "$lib/gen/api_pb";

const tokenKey = "token";

/**
 * authHeaders returns the request headers needed to authenticate Connect-RPC
 * calls using the same session token that the REST `ajax` helper uses.
 */
function authHeaders(): Record<string, string> {
  const token = typeof localStorage !== "undefined" ? localStorage.getItem(tokenKey) : null;
  if (token) {
    return { "X-Auth": token };
  }
  return {};
}

/**
 * transport is a Connect transport that targets the /connect/ prefix mounted
 * on the Gin server.  It injects the X-Auth session token on every request.
 */
const transport = createConnectTransport({
  baseUrl: "/connect",
  fetch: (input, init) =>
    fetch(input, {
      ...init,
      headers: {
        ...(init?.headers as Record<string, string> | undefined),
        ...authHeaders()
      }
    })
});

/**
 * paisaClient is the fully-typed Connect-RPC client for PaisaService.
 * Import and call its methods exactly like a regular async function.
 */
export const paisaClient: Client<typeof PaisaService> = createClient(PaisaService, transport);

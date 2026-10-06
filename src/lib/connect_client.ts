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
  fetch: async (input, init) => {
    const headers = new Headers(init?.headers);
    const auth = authHeaders();
    for (const [key, value] of Object.entries(auth)) {
      headers.set(key, value);
    }
    const res = await fetch(input, {
      ...init,
      headers
    });
    const contentEncoding = res.headers.get("content-encoding");
    if (contentEncoding && contentEncoding.includes("gzip")) {
      const arrayBuffer = await res.arrayBuffer();
      const bytes = new Uint8Array(arrayBuffer);
      if (bytes[0] === 0x1f && bytes[1] === 0x8b) {
        const ds = new DecompressionStream("gzip");
        const decompressedStream = new Response(bytes).body!.pipeThrough(ds);
        return new Response(decompressedStream, {
          status: res.status,
          statusText: res.statusText,
          headers: (() => {
            const h = new Headers(res.headers);
            h.delete("content-encoding");
            return h;
          })()
        });
      }
    }
    return res;
  }
});

/**
 * paisaClient is the fully-typed Connect-RPC client for PaisaService.
 * Import and call its methods exactly like a regular async function.
 */
export const paisaClient: Client<typeof PaisaService> = createClient(PaisaService, transport);

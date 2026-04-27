/**
 * account_tree.ts — typed account-tree fetch using the Connect-RPC client.
 *
 * This module demonstrates issue P3.1 #238: replacing a manual REST fetch
 * with a typed Connect client call that preserves the same loading / auth
 * behaviour as the existing `ajax` helper.
 *
 * The returned `AccountNode` tree is richer than the flat `string[]` returned
 * by `/api/config` — it reflects the colon-delimited hierarchy and marks
 * which nodes are leaf accounts (have actual postings).
 *
 * Usage:
 *   import { fetchAccountTree, flattenAccountTree } from "$lib/account_tree";
 *
 *   // Hierarchical (new capability via Connect):
 *   const tree = await fetchAccountTree();
 *
 *   // Flat list compatible with the existing accounts: string[] shape:
 *   const accounts = flattenAccountTree(tree);
 */

import { loading } from "../store";
import { paisaClient } from "$lib/connect_client";
import type { AccountNode } from "$lib/gen/api_pb";

export type { AccountNode };

/**
 * fetchAccountTree calls the typed `GetAccountTree` Connect endpoint and
 * returns the top-level `AccountNode` array.
 *
 * It sets the shared `loading` store to `true` while the request is in flight
 * (matching the behaviour of the existing `ajax` helper) unless `background`
 * is set to `true`.
 */
export async function fetchAccountTree(opts?: { background?: boolean }): Promise<AccountNode[]> {
  if (!opts?.background) {
    loading.set(true);
  }
  try {
    const response = await paisaClient.getAccountTree({});
    return response.accounts;
  } finally {
    if (!opts?.background) {
      loading.set(false);
    }
  }
}

/**
 * flattenAccountTree converts the hierarchical `AccountNode` tree into a flat
 * list of full account names, visiting only leaf nodes.  The result is
 * compatible with the `accounts: string[]` shape returned by `/api/config`.
 */
export function flattenAccountTree(nodes: AccountNode[]): string[] {
  const result: string[] = [];

  function visit(node: AccountNode) {
    if (node.isLeaf) {
      result.push(node.fullName);
    }
    for (const child of node.children) {
      visit(child);
    }
  }

  for (const node of nodes) {
    visit(node);
  }

  return result.sort();
}

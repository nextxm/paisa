/**
 * PersistedState — class-based rune adapter for persisted UI settings.
 *
 * Uses Svelte 5's `fromStore` to expose the existing Svelte stores in
 * `src/persisted_store.ts` as a unified, class-shaped reactive object.
 * Reading any property in a reactive context automatically tracks the
 * underlying store as a dependency; writing through the class propagates the
 * change back to the store (and thus to `localStorage`).
 *
 * ## Usage
 *
 * ### New-style (recommended)
 * ```svelte
 * <script>
 *   import { persistedState } from '$lib/state/persisted.svelte';
 * </script>
 * <input type="checkbox" bind:checked={persistedState.obscure} />
 * ```
 *
 * ### Legacy style (unchanged existing components)
 * ```svelte
 * <script>
 *   import { obscure } from '../../persisted_store';
 * </script>
 * <input type="checkbox" bind:checked={$obscure} />
 * ```
 */

import { fromStore } from "svelte/store";
import {
  obscure as obscureStore,
  cashflowExpenseDepthAllowed as cashflowExpenseDepthAllowedStore,
  cashflowExpenseDepth as cashflowExpenseDepthStore,
  cashflowIncomeDepthAllowed as cashflowIncomeDepthAllowedStore,
  cashflowIncomeDepth as cashflowIncomeDepthStore,
  cashflowShowTransfers as cashflowShowTransfersStore,
  sankeyPeriod as sankeyPeriodStore,
  sankeyRefDate as sankeyRefDateStore,
  type SankeyPeriod
} from "../../persisted_store";

// Re-export the type so consumers can import it from here too.
export type { SankeyPeriod };

// ─── Module-level fromStore bindings ─────────────────────────────────────────

const _obscure = fromStore(obscureStore);
const _cashflowExpenseDepthAllowed = fromStore(cashflowExpenseDepthAllowedStore);
const _cashflowExpenseDepth = fromStore(cashflowExpenseDepthStore);
const _cashflowIncomeDepthAllowed = fromStore(cashflowIncomeDepthAllowedStore);
const _cashflowIncomeDepth = fromStore(cashflowIncomeDepthStore);
const _cashflowShowTransfers = fromStore(cashflowShowTransfersStore);
const _sankeyPeriod = fromStore(sankeyPeriodStore);
const _sankeyRefDate = fromStore(sankeyRefDateStore);

// ─── PersistedState class ─────────────────────────────────────────────────────

/**
 * Provides class-based, rune-compatible access to the persisted settings.
 *
 * Persisted properties (obscure, cashflow depths, sankey settings) are
 * automatically written to `localStorage` by the underlying stores when they
 * change.  Non-persisted bookkeeping state (depth-allowed ranges, computed by
 * the API) lives here too for cohesion.
 */
class PersistedState {
  // ── obscure ───────────────────────────────────────────────────────────────

  get obscure(): boolean {
    return _obscure.current;
  }
  set obscure(v: boolean) {
    _obscure.current = v;
  }

  // ── cashflow expense depth ────────────────────────────────────────────────

  get cashflowExpenseDepthAllowed(): { min: number; max: number } {
    return _cashflowExpenseDepthAllowed.current;
  }
  set cashflowExpenseDepthAllowed(v: { min: number; max: number }) {
    _cashflowExpenseDepthAllowed.current = v;
  }

  get cashflowExpenseDepth(): number {
    return _cashflowExpenseDepth.current;
  }
  set cashflowExpenseDepth(v: number) {
    _cashflowExpenseDepth.current = v;
  }

  // ── cashflow income depth ─────────────────────────────────────────────────

  get cashflowIncomeDepthAllowed(): { min: number; max: number } {
    return _cashflowIncomeDepthAllowed.current;
  }
  set cashflowIncomeDepthAllowed(v: { min: number; max: number }) {
    _cashflowIncomeDepthAllowed.current = v;
  }

  get cashflowIncomeDepth(): number {
    return _cashflowIncomeDepth.current;
  }
  set cashflowIncomeDepth(v: number) {
    _cashflowIncomeDepth.current = v;
  }

  // ── cashflow show transfers ───────────────────────────────────────────────

  get cashflowShowTransfers(): boolean {
    return _cashflowShowTransfers.current;
  }
  set cashflowShowTransfers(v: boolean) {
    _cashflowShowTransfers.current = v;
  }

  // ── sankey period ─────────────────────────────────────────────────────────

  get sankeyPeriod(): SankeyPeriod {
    return _sankeyPeriod.current;
  }
  set sankeyPeriod(v: SankeyPeriod) {
    _sankeyPeriod.current = v;
  }

  // ── sankey ref date ───────────────────────────────────────────────────────

  get sankeyRefDate(): string {
    return _sankeyRefDate.current;
  }
  set sankeyRefDate(v: string) {
    _sankeyRefDate.current = v;
  }

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Update the allowed depth ranges after receiving API data, and clamp the
   * current depth values to the new valid range.
   */
  setCashflowDepthAllowed(expense: number, income: number): void {
    this.cashflowExpenseDepthAllowed = { min: 1, max: expense };
    if (this.cashflowExpenseDepth === 0 || this.cashflowExpenseDepth > expense) {
      this.cashflowExpenseDepth = expense;
    }

    this.cashflowIncomeDepthAllowed = { min: 1, max: income };
    if (this.cashflowIncomeDepth === 0 || this.cashflowIncomeDepth > income) {
      this.cashflowIncomeDepth = income;
    }
  }
}

// ─── Singleton export ─────────────────────────────────────────────────────────

export const persistedState = new PersistedState();

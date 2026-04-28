/**
 * UIState — class-based rune adapter for volatile (non-persisted) UI state.
 *
 * Uses Svelte 5's `fromStore` to expose the existing Svelte stores in
 * `src/store.ts` as a unified, class-shaped reactive object.  Reading any
 * property in a reactive context (component template, `$derived`, `$effect`)
 * automatically tracks the underlying store as a dependency and re-runs the
 * computation when the store value changes.
 *
 * ## Usage
 *
 * ### New-style (recommended for new components / refactored code)
 * ```svelte
 * <script>
 *   import { uiState } from '$lib/state/ui.svelte';
 * </script>
 * {uiState.loading ? 'Loading…' : ''}
 * ```
 *
 * ### Legacy style (unchanged existing components)
 * ```svelte
 * <script>
 *   import { loading } from '../../store';
 * </script>
 * {$loading ? 'Loading…' : ''}
 * ```
 *
 * Both styles are fully interoperable: writing via one path is immediately
 * reflected when reading via the other.
 */

import { fromStore } from "svelte/store";
import * as d3 from "d3";
import type { Dayjs } from "dayjs";
import type { AccountTfIdf } from "$lib/utils";
import {
  editorState as editorStateStore,
  sheetEditorState as sheetEditorStateStore,
  month as monthStore,
  year as yearStore,
  dateRangeOption as dateRangeOptionStore,
  dateMin as dateMinStore,
  dateMax as dateMaxStore,
  dateRange as dateRangeStore,
  theme as themeStore,
  loading as loadingStore,
  delayedLoading as delayedLoadingStore,
  delayedUnLoading as delayedUnLoadingStore,
  willClearTippy as willClearTippyStore,
  accountTfIdf as accountTfIdfStore,
  willRefresh as willRefreshStore,
  type EditorState,
  type SheetEditorState,
  initialEditorState,
  initialSheetEditorState
} from "../../store";

// Re-export types and constants so consumers can import everything from one place.
export type { EditorState, SheetEditorState };
export { initialEditorState, initialSheetEditorState };

// ─── Module-level fromStore bindings ─────────────────────────────────────────
//
// These are permanent bindings: because they are created at module scope
// (outside any component or effect), `fromStore` subscribes globally and never
// unsubscribes.  Any reactive context that reads `.current` will track the
// underlying store as a dependency.

const _editorState = fromStore(editorStateStore);
const _sheetEditorState = fromStore(sheetEditorStateStore);
const _month = fromStore(monthStore);
const _year = fromStore(yearStore);
const _dateRangeOption = fromStore(dateRangeOptionStore);
const _dateMin = fromStore(dateMinStore);
const _dateMax = fromStore(dateMaxStore);
const _dateRange = fromStore(dateRangeStore);
const _theme = fromStore(themeStore);
const _loading = fromStore(loadingStore);
const _delayedLoading = fromStore(delayedLoadingStore);
const _delayedUnLoading = fromStore(delayedUnLoadingStore);
const _willClearTippy = fromStore(willClearTippyStore);
const _accountTfIdf = fromStore(accountTfIdfStore);
const _willRefresh = fromStore(willRefreshStore);

// ─── UIState class ────────────────────────────────────────────────────────────

/**
 * Provides class-based, rune-compatible access to the shared UI state.
 * Every property getter/setter delegates to the corresponding Svelte store so
 * changes are always synchronised between legacy `$store` consumers and new
 * `uiState.*` consumers.
 */
class UIState {
  // ── editor state ──────────────────────────────────────────────────────────

  get editorState(): EditorState {
    return _editorState.current;
  }
  set editorState(v: EditorState) {
    _editorState.current = v;
  }

  get sheetEditorState(): SheetEditorState {
    return _sheetEditorState.current;
  }
  set sheetEditorState(v: SheetEditorState) {
    _sheetEditorState.current = v;
  }

  // ── date / period selectors ───────────────────────────────────────────────

  get month(): string {
    return _month.current;
  }
  set month(v: string) {
    _month.current = v;
  }

  get year(): string {
    return _year.current;
  }
  set year(v: string) {
    _year.current = v;
  }

  get dateRangeOption(): number {
    return _dateRangeOption.current;
  }
  set dateRangeOption(v: number) {
    _dateRangeOption.current = v;
  }

  get dateMin(): Dayjs {
    return _dateMin.current;
  }
  set dateMin(v: Dayjs) {
    _dateMin.current = v;
  }

  get dateMax(): Dayjs {
    return _dateMax.current;
  }
  set dateMax(v: Dayjs) {
    _dateMax.current = v;
  }

  /** Derived date range — read-only. */
  get dateRange(): { from: Dayjs; to: Dayjs } {
    return _dateRange.current;
  }

  // ── theme ─────────────────────────────────────────────────────────────────

  get theme(): string {
    return _theme.current;
  }
  set theme(v: string) {
    _theme.current = v;
  }

  // ── loading indicators ────────────────────────────────────────────────────

  get loading(): boolean {
    return _loading.current;
  }
  set loading(v: boolean) {
    _loading.current = v;
  }

  /** Delayed-show variant — read-only (derived from `loading`). */
  get delayedLoading(): boolean {
    return _delayedLoading.current;
  }

  /** Delayed-hide variant — read-only (derived from `loading`). */
  get delayedUnLoading(): boolean {
    return _delayedUnLoading.current;
  }

  // ── tippy / refresh signals ───────────────────────────────────────────────

  get willClearTippy(): number {
    return _willClearTippy.current;
  }
  set willClearTippy(v: number) {
    _willClearTippy.current = v;
  }

  get willRefresh(): number {
    return _willRefresh.current;
  }
  set willRefresh(v: number) {
    _willRefresh.current = v;
  }

  // ── TF-IDF index for account auto-complete ────────────────────────────────

  get accountTfIdf(): AccountTfIdf | null {
    return _accountTfIdf.current;
  }
  set accountTfIdf(v: AccountTfIdf | null) {
    _accountTfIdf.current = v;
  }

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Update `dateMin` / `dateMax` to span the supplied array of dates.
   * Called by page components after they receive data from the API.
   */
  setAllowedDateRange(dates: Dayjs[]): void {
    const [start, end] = d3.extent(dates);
    if (start) {
      this.dateMin = start;
      this.dateMax = end!;
    }
  }

  /**
   * Increment `willRefresh` to trigger a full page reload.
   * Guards against discarding unsaved editor changes.
   */
  refresh(): boolean {
    if (this.editorState.hasUnsavedChanges) {
      const confirmed = confirm("You have unsaved changes. Are you sure you want to leave?");
      if (!confirmed) return false;
      this.editorState = { ...this.editorState, hasUnsavedChanges: false };
    }
    this.willRefresh++;
    return true;
  }
}

// ─── Singleton export ─────────────────────────────────────────────────────────

export const uiState = new UIState();

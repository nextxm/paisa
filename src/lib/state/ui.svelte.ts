import { fromStore } from "svelte/store";
import {
  editorState,
  sheetEditorState,
  month,
  year,
  dateRangeOption,
  dateMin,
  dateMax,
  dateRange,
  theme,
  loading,
  delayedLoading,
  delayedUnLoading,
  willClearTippy,
  accountTfIdf,
  willRefresh
} from "../../store";

/**
 * Class-based reactive wrapper around the writable stores in store.ts.
 * Use `uiState.<property>.current` for reading and writing store values
 * in Svelte 5 rune-based components. The underlying `$store` syntax in
 * legacy components continues to work unchanged.
 */
class UIState {
  readonly editorState = fromStore(editorState);
  readonly sheetEditorState = fromStore(sheetEditorState);
  readonly month = fromStore(month);
  readonly year = fromStore(year);
  readonly dateRangeOption = fromStore(dateRangeOption);
  readonly dateMin = fromStore(dateMin);
  readonly dateMax = fromStore(dateMax);
  readonly dateRange = fromStore(dateRange);
  readonly theme = fromStore(theme);
  readonly loading = fromStore(loading);
  readonly delayedLoading = fromStore(delayedLoading);
  readonly delayedUnLoading = fromStore(delayedUnLoading);
  readonly willClearTippy = fromStore(willClearTippy);
  readonly accountTfIdf = fromStore(accountTfIdf);
  readonly willRefresh = fromStore(willRefresh);
}

export const uiState = new UIState();

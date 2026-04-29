<script lang="ts">
  import { now } from "$lib/utils";
  import dayjs from "dayjs";
  import _ from "lodash";

  let {
    min,
    max,
    value = $bindable()
  }: {
    min: dayjs.Dayjs;
    max: dayjs.Dayjs;
    value: string;
  } = $props();

  let open = $state(false);

  const valueDate = $derived(dayjs(value, "YYYY-MM"));
  let allowedYears = $derived(_.range(min.year(), max.year() + 1));
  let selectedYear = $state(valueDate.year());

  $effect(() => {
    if (!isAllowed(valueDate, min, max)) {
      if (isAllowed(now(), min, max)) {
        select(now());
      } else {
        select(max);
      }
    }
  });

  function isAllowed(date: dayjs.Dayjs, min: dayjs.Dayjs, max: dayjs.Dayjs) {
    return date.isSameOrAfter(min.startOf("month")) && date.isSameOrBefore(max.endOf("month"));
  }

  function select(date: dayjs.Dayjs) {
    value = date.format("YYYY-MM");
    selectedYear = date.year();
    open = false;
  }

  function selectMonth(month: number) {
    select(dayjs(`${selectedYear}-${month + 1}`, "YYYY-M"));
  }

  function selectYear(event: any) {
    selectedYear = parseInt(event.target.value);
  }

  const MONTHS = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec"
  ];
</script>

<div class="is-flex">
  <button
    class="button is-small border-left"
    aria-label="Previous month"
    disabled={!isAllowed(valueDate.add(-1, "month"), min, max)}
    onclick={(_e) => select(valueDate.add(-1, "month"))}
  >
    <span class="icon">
      <i class="fas fa-chevron-left"></i>
    </span>
  </button>
  <div class="dropdown is-right month-picker is-small" class:is-active={open}>
    <div class="dropdown-trigger">
      <button
        class="button is-small border-none"
        aria-haspopup="true"
        aria-controls="dropdown-menu2"
        onclick={(_e) => (open = !open)}
      >
        <span class="has-text-weight-bold">{valueDate.format("MMM YYYY")}</span>
        <span class="icon">
          <i class="fas fa-angle-down" aria-hidden="true"></i>
        </span>
      </button>
    </div>
    <div class="dropdown-menu" id="dropdown-menu2" role="menu">
      <div class="dropdown-content p-0">
        <div class="dropdown-item">
          <div class="is-flex is-justify-content-space-between is-align-items-center py-0 my-0">
            <button
              class="button is-small"
              aria-label="Previous year"
              disabled={selectedYear - 1 < min.year()}
              onclick={(_e) => selectedYear--}
            >
              <span class="icon">
                <i class="fas fa-chevron-left"></i>
              </span>
            </button>
            <div class="select">
              <select
                class="has-text-weight-bold"
                value={selectedYear}
                onchange={(e) => selectYear(e)}
              >
                {#each allowedYears as year}
                  <option value={year}>{year}</option>
                {/each}
              </select>
            </div>
            <button
              class="button is-small"
              aria-label="Next year"
              disabled={selectedYear + 1 > max.year()}
              onclick={(_e) => selectedYear++}
            >
              <span class="icon">
                <i class="fas fa-chevron-right"></i>
              </span>
            </button>
          </div>
        </div>
        <hr class="dropdown-divider m-0" />
        <div class="dropdown-item">
          <div class="months is-flex is-flex-wrap-wrap is-justify-content-space-between">
            {#each MONTHS as month, i}
              <div class="month is-size-6 py-2">
                {#if isAllowed(dayjs(`${selectedYear}-${i + 1}`, "YYYY-M"), min, max)}
                  <button
                    type="button"
                    class={valueDate.year() == selectedYear && valueDate.month() == i
                      ? "is-link has-text-weight-bold"
                      : "has-text-black-ter"}
                    aria-label={`Select ${month} ${selectedYear}`}
                    onclick={(_e) => selectMonth(i)}
                  >
                    {month}
                  </button>
                {:else}
                  <span class="has-text-grey-light">{month}</span>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>
  </div>
  <button
    class="button is-small border-right"
    aria-label="Next month"
    disabled={!isAllowed(valueDate.add(1, "month"), min, max)}
    onclick={(_e) => select(valueDate.add(1, "month"))}
  >
    <span class="icon">
      <i class="fas fa-chevron-right"></i>
    </span>
  </button>
</div>

<style lang="scss">
  .button,
  .select select {
    border: none;
    box-shadow: none;

    &:hover,
    &:focus {
      border: none;
      box-shadow: none;
      outline: none;
    }
  }

  .month button {
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
    transition: background-color 120ms ease;
    cursor: pointer;

    &.has-text-black-ter {
      color: hsl(0, 0%, 29%);

      &:hover,
      &:focus {
        background-color: hsl(0, 0%, 96%);
        outline: 1px solid hsl(0, 0%, 86%);
        outline-offset: -1px;
      }
    }

    &.is-link {
      color: hsl(217, 71%, 53%);
    }
  }
</style>

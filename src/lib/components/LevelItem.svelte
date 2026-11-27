<script lang="ts">
  import _ from "lodash";

  let {
    title,
    value,
    small = false,
    narrow = false,
    color = null,
    subtitle = null,
    href = null
  }: {
    title: string;
    value: string;
    small?: boolean;
    narrow?: boolean;
    color?: string;
    subtitle?: string;
    href?: string | null;
  } = $props();
</script>

<div class="level-item {narrow && 'is-narrow'} has-text-left" class:small>
  <div>
    <p class="heading">{title}</p>
    {#if href}
      <a {href} class="is-block">
        {#if color}
          <p class="title" style="padding: 5px; color: {color};">{value}</p>
        {:else}
          <p class="title has-text-grey-dark">{value}</p>
        {/if}
      </a>
    {:else if color}
      <p class="title" style="padding: 5px; color: {color};">{value}</p>
    {:else}
      <p class="title has-text-grey-dark">{value}</p>
    {/if}
    {#if !_.isEmpty(subtitle)}
      <div class="sub-title">{@html subtitle}</div>
    {/if}
  </div>
</div>

<style lang="scss">
  @import "bulma/sass/utilities/_all.sass";

  .level-item.small {
    .title {
      font-size: 1.25rem !important;
      line-height: 1.5rem !important;
    }
  }

  .heading {
    font-weight: 600 !important;
    font-size: 0.75rem !important;
    text-transform: uppercase !important;
    letter-spacing: 0.04em !important;
    opacity: 0.72;
    margin-bottom: 0.2rem !important;
  }

  .title {
    font-weight: 700 !important;
    font-size: 1.65rem !important;
    line-height: 1.9rem !important;
    font-variant-numeric: tabular-nums;
    padding-left: 0 !important;
    padding-bottom: 0 !important;
    margin-bottom: 0 !important;
  }

  @include mobile {
    .title {
      font-size: 1.45rem !important;
      line-height: 1.75rem !important;
      word-break: break-word;
    }

    .sub-title {
      font-size: 0.85rem !important;
    }
  }

  @include widescreen {
    .title {
      font-size: 2.25rem !important;
      line-height: 2.5rem !important;
    }

    .level-item.small {
      .title {
        font-size: 1.5rem !important;
        line-height: 1.75rem !important;
      }
    }
  }

  .sub-title {
    font-weight: normal !important;
    font-size: 0.75rem !important;
  }
</style>

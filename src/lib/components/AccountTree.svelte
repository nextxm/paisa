<!--
  AccountTree.svelte — hierarchical account tree with expand/collapse and selection.

  Uses Svelte 5 runes for state and derived computations.
  Accepts an `AccountNode[]` tree (from fetchAccountTree / api_pb.ts) and
  exposes the selected account name via the `selected` bindable prop.

  Props:
    nodes      – AccountNode array to render at this level
    selected   – currently selected full account name (bindable)
    depth      – current nesting depth (used for recursive rendering, default 0)
-->
<script lang="ts">
  import type { AccountNode } from "$lib/gen/api_pb";
  import AccountTree from "./AccountTree.svelte";

  let {
    nodes = [],
    selected = $bindable(""),
    depth = 0
  }: {
    nodes?: AccountNode[];
    selected?: string;
    depth?: number;
  } = $props();

  // Track which nodes are expanded. Depth 0 and 1 are expanded by default.
  let expanded: Record<string, boolean> = $state({});

  $effect(() => {
    for (const n of nodes) {
      if (!(n.fullName in expanded)) {
        expanded[n.fullName] = depth < 2;
      }
    }
  });

  function toggle(node: AccountNode) {
    expanded[node.fullName] = !expanded[node.fullName];
  }

  function select(node: AccountNode) {
    selected = node.fullName;
  }

  function handleKeydown(e: KeyboardEvent, node: AccountNode) {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      if (node.children.length > 0) {
        toggle(node);
      } else {
        select(node);
      }
    }
  }
</script>

<ul class="account-tree" class:account-tree-root={depth === 0}>
  {#each nodes as node (node.fullName)}
    {@const isExpanded = expanded[node.fullName] ?? depth < 2}
    {@const isSelected = selected === node.fullName}
    {@const hasChildren = node.children.length > 0}
    <li class="account-tree-node">
      <div
        class="account-tree-row flex items-center gap-1 px-1 py-0.5 rounded cursor-pointer select-none
          {isSelected ? 'bg-primary/10 text-primary font-semibold' : 'hover:bg-base-200'}"
        style="padding-left: {depth * 1}rem"
        role="treeitem"
        aria-selected={isSelected}
        aria-expanded={hasChildren ? isExpanded : undefined}
        tabindex="0"
        onclick={() => {
          if (hasChildren) toggle(node);
          else select(node);
        }}
        onkeydown={(e) => handleKeydown(e, node)}
      >
        <!-- expand/collapse indicator -->
        <span class="account-tree-toggle w-4 text-center shrink-0">
          {#if hasChildren}
            <i
              class="fas fa-angle-{isExpanded ? 'down' : 'right'} text-xs text-base-content/60"
              aria-hidden="true"
            ></i>
          {:else}
            <span class="text-xs text-base-content/30">·</span>
          {/if}
        </span>

        <!-- node label -->
        <span class="account-tree-label text-sm truncate" title={node.fullName}>
          {node.name}
        </span>
      </div>

      {#if hasChildren && isExpanded}
        <AccountTree bind:selected depth={depth + 1} nodes={node.children} />
      {/if}
    </li>
  {/each}
</ul>

<style lang="scss">
  .account-tree-root {
    padding: 0;
    margin: 0;
    list-style: none;
  }

  .account-tree {
    padding: 0;
    margin: 0;
    list-style: none;
  }

  .account-tree-node {
    line-height: 1.6;
  }

  .account-tree-row {
    &:focus-visible {
      outline: 2px solid hsl(var(--p));
      outline-offset: -1px;
    }
  }
</style>

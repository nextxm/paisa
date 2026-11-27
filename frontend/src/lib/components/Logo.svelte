<script lang="ts">
  let {
    size = 24,
    thickness = 4,
    animation = false
  }: { size?: number; thickness?: number; animation?: boolean } = $props();

  const colors = ["#00c853", "#ff5252", "#ffab00", "#d500f9", "#00b0ff"];
  const dotColor = "#6366f1";
</script>

<svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24">
  <g>
    <g transform="rotate(360, 12, 12)">
      <path
        d="M12 12Z"
        fill="none"
        stroke={dotColor}
        stroke-width={thickness}
        stroke-linecap="round"
      />
    </g>
    <g>
      {#each colors as color, i}
        <g transform="rotate({i * 72}, 12, 12)">
          <path
            d="M 21.510565162951536 8.909830056250526 A 10 10 0 0 0 12 6"
            fill="none"
            stroke={color}
            stroke-width={thickness}
          />
        </g>
      {/each}

      {#each colors as color, i}
        <g transform="rotate({(i + 1) * 72}, 12, 12)">
          <path
            d="M12.59 5.88Z"
            fill="none"
            stroke={color}
            stroke-width={thickness * 1.05}
            stroke-linecap="round"
          />
        </g>
      {/each}

      {#if animation}
        <animateTransform
          attributeName="transform"
          attributeType="XML"
          type="rotate"
          from="0 12 12"
          to="360 12 12"
          dur="1s"
          repeatCount="indefinite"
        />
      {/if}
    </g>
  </g>
</svg>

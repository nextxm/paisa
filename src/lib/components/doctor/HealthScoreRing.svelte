<script lang="ts">
  let {
    score = 100,
    size = 110,
    strokeWidth = 10
  }: {
    score?: number;
    size?: number;
    strokeWidth?: number;
  } = $props();

  const radius = $derived((size - strokeWidth) / 2);
  const circumference = $derived(2 * Math.PI * radius);
  const strokeDashoffset = $derived(circumference - (score / 100) * circumference);

  const ringColor = $derived.by(() => {
    if (score >= 90) return "var(--health-ring-good, #48c78e)";
    if (score >= 70) return "var(--health-ring-warn, #ffe08a)";
    return "var(--health-ring-danger, #f14668)";
  });
</script>

<div class="health-ring-wrapper" style="width: {size}px; height: {size}px;">
  <svg width={size} height={size} viewBox="0 0 {size} {size}" class="health-ring-svg">
    <!-- Track Circle -->
    <circle
      cx={size / 2}
      cy={size / 2}
      r={radius}
      stroke="rgba(255, 255, 255, 0.08)"
      stroke-width={strokeWidth}
      fill="none"
    />
    <!-- Progress Circle -->
    <circle
      cx={size / 2}
      cy={size / 2}
      r={radius}
      stroke={ringColor}
      stroke-width={strokeWidth}
      stroke-dasharray={circumference}
      stroke-dashoffset={strokeDashoffset}
      stroke-linecap="round"
      fill="none"
      class="health-ring-progress"
    />
  </svg>
  <div class="health-ring-center">
    <span class="health-score-value">{score}%</span>
    <span class="health-score-label">HEALTH</span>
  </div>
</div>

<style>
  .health-ring-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .health-ring-svg {
    transform: rotate(-90deg);
  }

  .health-ring-progress {
    transition:
      stroke-dashoffset 0.6s cubic-bezier(0.4, 0, 0.2, 1),
      stroke 0.4s ease;
  }

  .health-ring-center {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    pointer-events: none;
  }

  .health-score-value {
    font-size: 1.4rem;
    font-weight: 800;
    line-height: 1;
    letter-spacing: -0.02em;
  }

  .health-score-label {
    font-size: 0.58rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    opacity: 0.6;
    margin-top: 2px;
  }
</style>

<script lang="ts">
  import { formatCurrency, formatFloat, type DrawdownResponse } from "$lib/utils";

  let { response }: { response: DrawdownResponse | null } = $props();

  function recommendationTaxTotal(
    recommendation: DrawdownResponse["drawdown"]["recommendations"][number]
  ) {
    return (
      recommendation.estimated_tax.short_term +
      recommendation.estimated_tax.long_term +
      recommendation.estimated_tax.slab
    );
  }

  function totalTax() {
    if (!response) return 0;
    return (
      response.drawdown.total_estimated_tax.short_term +
      response.drawdown.total_estimated_tax.long_term +
      response.drawdown.total_estimated_tax.slab
    );
  }
</script>

{#if response}
  <div class="drawdown-layout">
    <div class="columns is-multiline mb-2">
      <div class="column is-4">
        <div class="metric-card">
          <div class="metric-label">Requested</div>
          <div class="metric-value">{formatCurrency(response.drawdown.requested_amount)}</div>
        </div>
      </div>
      <div class="column is-4">
        <div class="metric-card">
          <div class="metric-label">Covered</div>
          <div class="metric-value">{formatCurrency(response.drawdown.recommended_amount)}</div>
        </div>
      </div>
      <div class="column is-4">
        <div class="metric-card">
          <div class="metric-label">Estimated Tax</div>
          <div class="metric-value">{formatCurrency(totalTax())}</div>
        </div>
      </div>
      <div class="column is-4">
        <div class="metric-card">
          <div class="metric-label">Uncovered</div>
          <div class="metric-value">{formatCurrency(response.drawdown.remaining_amount)}</div>
        </div>
      </div>
    </div>

    {#if response.drawdown.recommendations.length === 0}
      <div class="notification is-warning is-light mb-0">
        No eligible lots matched the current strategy buckets.
      </div>
    {:else}
      <div class="strategy-list">
        {#each response.drawdown.recommendations as recommendation, index (`${recommendation.account}-${index}`)}
          <div class="strategy-card">
            <div class="strategy-header">
              <div>
                <div class="strategy-title">{index + 1}. {recommendation.account}</div>
                <div class="strategy-subtitle">
                  {recommendation.commodity} · {recommendation.tax_category}
                </div>
              </div>
              <div class="strategy-amount">{formatCurrency(recommendation.amount)}</div>
            </div>

            <div class="strategy-grid">
              <div>
                <div class="metric-label">Units</div>
                <div>{formatFloat(recommendation.units)}</div>
              </div>
              <div>
                <div class="metric-label">Holding Period</div>
                <div>{recommendation.holding_period_months} months</div>
              </div>
              <div>
                <div class="metric-label">Purchase Date</div>
                <div>{recommendation.purchase_date.format("YYYY-MM-DD")}</div>
              </div>
              <div>
                <div class="metric-label">Tax Rate</div>
                <div>{formatFloat(recommendation.effective_tax_rate * 100)}%</div>
              </div>
              <div>
                <div class="metric-label">Tax</div>
                <div>{formatCurrency(recommendationTaxTotal(recommendation))}</div>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .metric-card,
  .strategy-card {
    padding: 0.85rem;
    border-radius: 8px;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.03));
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }

  .metric-label {
    font-size: 0.72rem;
    font-weight: 600;
    letter-spacing: 0.03em;
    opacity: 0.65;
    text-transform: uppercase;
  }

  .metric-value {
    font-size: 1.2rem;
    font-weight: 700;
    margin-top: 0.25rem;
  }

  .strategy-list {
    display: grid;
    gap: 0.75rem;
  }

  .strategy-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 0.75rem;
  }

  .strategy-title {
    font-weight: 700;
  }

  .strategy-subtitle {
    font-size: 0.8rem;
    opacity: 0.7;
    margin-top: 0.15rem;
  }

  .strategy-amount {
    font-size: 1.1rem;
    font-weight: 700;
    white-space: nowrap;
  }

  .strategy-grid {
    display: grid;
    gap: 0.75rem;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  }

  @media (max-width: 768px) {
    .strategy-header {
      flex-direction: column;
    }
  }
</style>

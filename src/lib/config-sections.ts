export interface ConfigSection {
  id: string;
  label: string;
  icon: string;
  schemaKeys: string[];
  description: string;
}

export interface ConfigGroup {
  label: string;
  sections: ConfigSection[];
}

export const CONFIG_GROUPS: ConfigGroup[] = [
  {
    label: "General",
    sections: [
      {
        id: "core-setup",
        label: "Core Setup",
        icon: "fa-gear",
        schemaKeys: ["journal_path", "db_path", "sheets_directory", "ledger_cli"],
        description: "Journal paths, ledger engine, and fundamental options"
      },
      {
        id: "display-format",
        label: "Display & Format",
        icon: "fa-palette",
        schemaKeys: [
          "default_currency",
          "currencies",
          "display_precision",
          "locale",
          "time_zone",
          "financial_year_starting_month",
          "week_starting_day",
          "amount_alignment_column"
        ],
        description: "Currency, locale, number formatting, and calendar preferences"
      },
      {
        id: "security",
        label: "Security & Auth",
        icon: "fa-shield-halved",
        schemaKeys: ["user_accounts", "strict"],
        description: "User accounts, passwords, and strict mode"
      }
    ]
  },
  {
    label: "Finance",
    sections: [
      {
        id: "commodities",
        label: "Commodities & Prices",
        icon: "fa-chart-line",
        schemaKeys: ["commodities"],
        description: "External price providers for stocks, mutual funds, and other assets"
      },
      {
        id: "allocation-targets",
        label: "Allocation Targets",
        icon: "fa-chart-pie",
        schemaKeys: ["allocation_targets"],
        description: "Define your target asset allocation to track portfolio drift"
      },
      {
        id: "credit-cards",
        label: "Credit Cards",
        icon: "fa-credit-card",
        schemaKeys: ["credit_cards"],
        description: "Credit card limits, statement cycles, and due dates"
      },
      {
        id: "schedule-al",
        label: "Schedule AL",
        icon: "fa-file-lines",
        schemaKeys: ["schedule_al"],
        description: "Map accounts to Indian Income Tax Schedule AL categories"
      }
    ]
  },
  {
    label: "Planning",
    sections: [
      {
        id: "goals",
        label: "Goals",
        icon: "fa-flag",
        schemaKeys: ["goals"],
        description: "Retirement savings targets and financial milestones"
      },
      {
        id: "budget",
        label: "Budget",
        icon: "fa-wallet",
        schemaKeys: ["budget"],
        description: "Monthly budget rollover and spending rules"
      }
    ]
  },
  {
    label: "Tools",
    sections: [
      {
        id: "import-templates",
        label: "Import Templates",
        icon: "fa-file-import",
        schemaKeys: ["import_templates"],
        description: "Templates for parsing CSV and bank statement imports"
      },
      {
        id: "accounts",
        label: "Accounts",
        icon: "fa-landmark",
        schemaKeys: ["accounts", "checking_accounts", "inactive_accounts", "enable_reconciliation"],
        description: "Custom icons and display settings for account names"
      },
      {
        id: "doctor",
        label: "Doctor Rules",
        icon: "fa-stethoscope",
        schemaKeys: ["doctor"],
        description: "Automated journal health checks for balances and prices"
      },
      {
        id: "firefly",
        label: "Firefly III",
        icon: "fa-fire",
        schemaKeys: ["firefly"],
        description: "Configure Firefly III integration for reconciliation"
      },
      {
        id: "labs",
        label: "Labs",
        icon: "fa-flask",
        schemaKeys: ["labs"],
        description: "Enable experimental features"
      },
      {
        id: "advanced",
        label: "Advanced",
        icon: "fa-sliders",
        schemaKeys: ["provider_debug_http", "disable_multi_currency_prices"],
        description: "Debug logging and experimental feature flags"
      }
    ]
  }
];

export const ALL_SECTIONS: ConfigSection[] = CONFIG_GROUPS.flatMap((g) => g.sections);
export const DEFAULT_SECTION_ID = "core-setup";

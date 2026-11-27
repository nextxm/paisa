export interface ConfigSection {
  id: string;
  label: string;
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
        schemaKeys: [
          "journal_path",
          "add_journal_path",
          "db_path",
          "sheets_directory",
          "ledger_cli"
        ],
        description: "Journal paths, ledger engine, and fundamental options"
      },
      {
        id: "display-format",
        label: "Display & Format",
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
        schemaKeys: ["commodities"],
        description: "External price providers for stocks, mutual funds, and other assets"
      },
      {
        id: "allocation-targets",
        label: "Allocation Targets",
        schemaKeys: ["allocation_targets"],
        description: "Define your target asset allocation to track portfolio drift"
      },
      {
        id: "credit-cards",
        label: "Credit Cards",
        schemaKeys: ["credit_cards"],
        description: "Credit card limits, statement cycles, and due dates"
      },
      {
        id: "schedule-al",
        label: "Schedule AL",
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
        schemaKeys: ["goals"],
        description: "Retirement savings targets and financial milestones"
      },
      {
        id: "budget",
        label: "Budget",
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
        schemaKeys: ["import_templates"],
        description: "Templates for parsing CSV and bank statement imports"
      },
      {
        id: "accounts",
        label: "Accounts",
        schemaKeys: ["accounts", "checking_accounts", "inactive_accounts"],
        description: "Custom icons and display settings for account names"
      },
      {
        id: "doctor",
        label: "Doctor Rules",
        schemaKeys: ["doctor"],
        description: "Automated journal health checks for balances and prices"
      },
      {
        id: "backup",
        label: "Backup & Export",
        schemaKeys: [],
        description: "Export full zip archive of journals, configuration, and sheet files"
      },
      {
        id: "firefly",
        label: "Firefly III",
        schemaKeys: ["firefly"],
        description: "Configure Firefly III integration for reconciliation"
      },
      {
        id: "labs",
        label: "Labs",
        schemaKeys: ["labs"],
        description: "Enable experimental features"
      },
      {
        id: "advanced",
        label: "Advanced",
        schemaKeys: [
          "provider_debug_http",
          "disable_multi_currency_prices",
          "disable_missing_price_logging",
          "use_legacy_ui"
        ],
        description: "Debugging, multi-currency flags, and UI preferences"
      }
    ]
  }
];

export const ALL_SECTIONS = CONFIG_GROUPS.flatMap((g) => g.sections);
export const DEFAULT_SECTION_ID = "core-setup";

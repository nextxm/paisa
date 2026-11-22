---
description: "Keyboard shortcuts supported by Paisa's Editor"
---

# Editor

Open **Ledger → Editor** (`/ledger/editor`) to edit your main journal and any included files in the same directory tree (or jump straight to a specific transaction line with `/ledger/editor/<file>#<line>` from the Transactions and Postings views).

Paisa uses [CodeMirror](https://codemirror.net/) and supports the [standard keymap](https://codemirror.net/docs/ref/#commands.standardKeymap) along with Paisa-specific features and layout controls:

- **Resizable & collapsible sidebars**: Toggle the left file tree and right validation output pane from the toolbar, or drag their resize handles (panel widths and collapsed states are saved locally in your browser).
- **Live validation & Prettify**: Edits are validated continuously against your configured `ledger_cli` engine with inline error diagnostics, an error count jump button, and automatic column alignment (`Ctrl+Shift+I` / **Prettify**) based on `amount_alignment_column` in [Configuration](./config.md).
- **Autocompletion & recurring schedule preview**: Suggests known accounts, payees, commodities, and today's date (`20...`), and renders the next 3 scheduled dates inline next to `; Period:` metadata tags.
- **File backups & revert**: Revert to timestamped `.backup.*` versions or clean up old backups directly from the editor toolbar.
- **Explicit sync after saving**: Saving writes the validated file to disk and flags the journal as needing a sync (`Saved ... Please sync to see changes`); click **Sync** in the top bar when you are ready to refresh reports.

## Keyboard shortcuts

| Linux            | Mac              | Windows          | Description               |
|------------------|------------------|------------------|---------------------------|
| ++ctrl+s++       | ++ctrl+s++       | ++ctrl+s++       | Save file                 |
| ++ctrl+shift+i++ | ++ctrl+shift+i++ | ++ctrl+shift+i++ | Prettify / align columns  |
| ++ctrl+f++       | ++cmd+f++        | ++ctrl+f++       | Open search panel         |
| ++ctrl+z++       | ++cmd+z++        | ++ctrl+z++       | Undo                      |
| ++ctrl+y++       | ++cmd+y++        | ++ctrl+y++       | Redo                      |


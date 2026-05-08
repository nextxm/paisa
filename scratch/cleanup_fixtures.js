const fs = require('fs');
const path = require('path');

function walk(dir, callback) {
    if (!fs.existsSync(dir)) return;
    fs.readdirSync(dir).forEach( f => {
        let dirPath = path.join(dir, f);
        let isDirectory = fs.statSync(dirPath).isDirectory();
        isDirectory ?
            walk(dirPath, callback) : callback(path.join(dir, f));
    });
};

walk('tests/fixture', (filePath) => {
    if (path.basename(filePath) === 'config.json' || path.basename(filePath) === 'paisa.yaml') {
        console.log(`Processing ${filePath}`);
        let content = fs.readFileSync(filePath, 'utf8');
        
        if (filePath.endsWith('.json')) {
            // First pass: remove the dangling blocks from my previous failed attempt
            content = content.replace(/"default": false,\s+"description": "Enable reconciliation feature[\s\S]+?"type": "boolean"\s+},/g, '');
            // Second pass: remove any existing whole blocks
            content = content.replace(/"enable_reconciliation":\s*{[\s\S]+?},/g, '');
            content = content.replace(/"enable_reconciliation":\s*false,?/g, '');
            
            // Clean up any remaining descriptions that mention reconciliation
            content = content.replace(/"description": "Accounts to ignore from reconciliation"/g, '"description": "Accounts to ignore"');
            
            // Try to format it back nicely
            try {
                let json = JSON.parse(content);
                fs.writeFileSync(filePath, JSON.stringify(json, null, 2));
            } catch (e) {
                console.log(`Cleaning up broken JSON for ${filePath}`);
                // If it's still broken (e.g. trailing commas), we might need more cleanup
                // But the regex above should handle the most common cases.
                fs.writeFileSync(filePath, content);
            }
        } else if (filePath.endsWith('.yaml')) {
            content = content.replace(/enable_reconciliation:.*\n/g, '');
            fs.writeFileSync(filePath, content);
        }
    }
});

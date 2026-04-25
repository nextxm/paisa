import fs from 'fs';
import path from 'path';

const version = process.env.VERSION;
if (!version) {
    console.error('VERSION environment variable is required');
    process.exit(1);
}

// Clean the version (remove 'v' prefix if present)
const cleanVersion = version.startsWith('v') ? version.substring(1) : version;
const date = new Date().toISOString().split('T')[0];

const changelogPath = path.resolve('CHANGELOG.md');
const versionGoPath = path.resolve('cmd/version.go');

// 1. Update CHANGELOG.md
if (fs.existsSync(changelogPath)) {
    let changelog = fs.readFileSync(changelogPath, 'utf8');
    const unreleasedHeader = /### Unreleased — (.*)/;
    const match = changelog.match(unreleasedHeader);

    if (match) {
        const description = match[1];
        const newHeader = `### ${cleanVersion} (${date}) — ${description}`;
        
        // Rotate: Rename Unreleased to Version, and prepend a new empty Unreleased section
        changelog = changelog.replace(unreleasedHeader, `### Unreleased — Future changes\n\n${newHeader}`);
        fs.writeFileSync(changelogPath, changelog);
        console.log(`Updated CHANGELOG.md to version ${cleanVersion}`);
    } else {
        console.warn('Could not find "### Unreleased" header in CHANGELOG.md');
    }
}

// 2. Update cmd/version.go
if (fs.existsSync(versionGoPath)) {
    let versionGo = fs.readFileSync(versionGoPath, 'utf8');
    const versionRegex = /var Version = "([^"]+)"/;
    versionGo = versionGo.replace(versionRegex, `var Version = "${cleanVersion}"`);
    fs.writeFileSync(versionGoPath, versionGo);
    console.log(`Updated cmd/version.go to version ${cleanVersion}`);
}

import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const generatedFiles = [
  'apps/admin/schema/openapi.json',
  'apps/admin/src/services/nexus',
];

function snapshot() {
  const files = new Map();
  const visit = (path) => {
    const entries = readdirSync(path, { withFileTypes: true }).sort((a, b) =>
      a.name.localeCompare(b.name),
    );
    for (const entry of entries) {
      const child = join(path, entry.name);
      if (entry.isDirectory()) {
        visit(child);
      } else {
        files.set(relative(root, child), readFileSync(child));
      }
    }
  };

  for (const path of generatedFiles) {
    const absolute = join(root, path);
    if (path.endsWith('.json')) {
      files.set(path, readFileSync(absolute));
    } else {
      visit(absolute);
    }
  }
  return files;
}

function generate() {
  const result = spawnSync('pnpm generate:contracts', {
    cwd: root,
    stdio: 'inherit',
    shell: true,
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

// A fresh clone has no generated files because they are intentionally ignored.
if (generatedFiles.some((path) => !existsSync(join(root, path)))) {
  generate();
}
const before = snapshot();
generate();
const after = snapshot();
const changed = [...new Set([...before.keys(), ...after.keys()])]
  .filter((path) => {
    const previous = before.get(path);
    const current = after.get(path);
    return !previous || !current || !previous.equals(current);
  })
  .sort();

if (changed.length > 0) {
  console.error('Generated contracts are out of date:');
  for (const path of changed) {
    console.error(`- ${path}`);
  }
  process.exit(1);
}

console.log('Generated OpenAPI and Admin clients are up to date.');

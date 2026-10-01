import {readFile, readdir} from 'node:fs/promises';
import {join, relative, resolve, sep} from 'node:path';

const root = resolve(import.meta.dirname, '..');
const allowedFiles = new Set([
  '.env.example', '.gitignore', 'README.md', 'go.mod', 'go.sum',
  'package.json', 'package-lock.json', 'tsconfig.browser.json',
  '.github/workflows/server-ci.yml', 'contracts/client-support.json',
  'contracts/http-v1.json', 'infra/Caddyfile', 'infra/compose.yaml',
  'tests/browser.e2e.ts', 'scripts/check-public-boundary.mjs',
]);
const allowedPrefixes = ['apps/server/'];
const excludedDirectories = new Set(['.git', 'node_modules', 'dist', 'coverage', 'test-results', '.ci-cli']);
const secretPatterns = [
  /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/,
  /(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{30,}/,
  /AKIA[0-9A-Z]{16}/,
  /npm_[A-Za-z0-9]{30,}/,
];

const failures = [];
async function visit(directory) {
  for (const entry of await readdir(directory, {withFileTypes: true})) {
    if (entry.isDirectory() && excludedDirectories.has(entry.name)) continue;
    const full = join(directory, entry.name);
    const path = relative(root, full).split(sep).join('/');
    if (entry.isSymbolicLink()) { failures.push(`symlink: ${path}`); continue; }
    if (entry.isDirectory()) { await visit(full); continue; }
    if (!entry.isFile() || !(allowedFiles.has(path) || allowedPrefixes.some(prefix => path.startsWith(prefix)))) {
      failures.push(`forbidden path: ${path}`);
      continue;
    }
    const content = await readFile(full, 'utf8');
    if (secretPatterns.some(pattern => pattern.test(content))) failures.push(`secret-like content: ${path}`);
  }
}
await visit(root);
if (failures.length) {
  for (const failure of failures) process.stderr.write(`${failure}\n`);
  process.exitCode = 1;
} else {
  process.stdout.write('Public server source boundary passed.\n');
}

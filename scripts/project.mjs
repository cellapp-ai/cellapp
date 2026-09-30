import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const [action, ...args] = process.argv.slice(2);
if (!['cli', 'typecheck', 'test', 'build'].includes(action)) {
  process.stderr.write('Unknown project command. Use cli, typecheck, test or build.\n');
  process.exit(2);
}
if (!existsSync(join(root, 'skills', '.git')) || !existsSync(join(root, 'skills', 'package.json'))) {
  process.stderr.write('Client submodule is not initialized. Run: git submodule update --init --recursive\n');
  process.exit(1);
}
function run(command, argv) {
  const result = spawnSync(command, argv, {cwd: root, stdio: 'inherit'});
  if (result.error) {
    process.stderr.write(`${result.error.message}\n`);
    process.exit(1);
  }
  if (result.status !== 0) process.exit(result.status ?? 1);
}
run('npm', ['--prefix', 'skills', 'run', action, ...(action === 'cli' ? ['--', ...args] : [])]);
if (action === 'typecheck') {
  run(process.execPath, ['node_modules/typescript/bin/tsc', '-p', 'tsconfig.json', '--noEmit']);
  run('npm', ['run', 'typecheck', '--workspace', '@cellapp/web']);
}
if (action === 'test') {
  run(process.execPath, ['--test', 'tests/project.test.mjs']);
  run('npm', ['run', 'build', '--workspace', '@cellapp/web']);
  run('go', ['test', './apps/server/...']);
  run('npm', ['run', 'lint', '--workspace', '@cellapp/web']);
  run('npm', ['test', '--workspace', '@cellapp/web']);
}
if (action === 'build') {
  mkdirSync(join(root, 'dist'), {recursive: true});
  run('npm', ['run', 'build', '--workspace', '@cellapp/web']);
  run('go', ['build', '-o', 'dist/cellapp-server', './apps/server/cmd/server']);
  rmSync(join(root, 'dist', 'web'), {recursive: true, force: true});
  cpSync(join(root, 'apps', 'web', 'dist'), join(root, 'dist', 'web'), {recursive: true});
}

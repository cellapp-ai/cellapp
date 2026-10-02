import {existsSync} from 'node:fs';
import {spawnSync} from 'node:child_process';

if (!existsSync('skills/packages/cli/src/client.ts')) {
  process.stdout.write('Skipping browser typecheck; client submodule is not initialized.\n');
  process.exit(0);
}
const result = spawnSync(process.execPath, ['node_modules/typescript/bin/tsc', '-p', 'tsconfig.browser.json', '--noEmit'], {stdio: 'inherit'});
process.exit(result.status ?? 1);

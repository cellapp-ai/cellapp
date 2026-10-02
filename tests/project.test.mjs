import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, copyFileSync, writeFileSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

test('project commands explain missing submodule and preserve CLI arguments and exit codes', () => {
  const root = mkdtempSync(join(tmpdir(), 'cellapp-project-command-'));
  try {
    mkdirSync(join(root, 'scripts'));
    copyFileSync(new URL('../scripts/project.mjs', import.meta.url), join(root, 'scripts/project.mjs'));
    const command = (...args) => spawnSync(process.execPath, [join(root, 'scripts/project.mjs'), ...args], {encoding: 'utf8', env: {...process.env, PATH: `${join(root, 'bin')}:${process.env.PATH}`, COMMAND_RECORD: join(root, 'record.json')}});
    const missing = command('cli', '--help');
    assert.equal(missing.status, 1);
    assert.match(missing.stderr, /git submodule update --init --recursive/);
    mkdirSync(join(root, 'skills', '.git'), {recursive: true});
    writeFileSync(join(root, 'skills', 'package.json'), '{}');
    mkdirSync(join(root, 'bin'));
    writeFileSync(join(root, 'bin', 'npm'), `#!${process.execPath}\nrequire('node:fs').writeFileSync(process.env.COMMAND_RECORD, JSON.stringify(process.argv.slice(2))); process.exit(23);\n`, {mode: 0o755});
    const result = command('cli', 'deploy', '--project', '/path with spaces', '--build', 'npm run build');
    assert.equal(result.status, 23);
    assert.deepEqual(JSON.parse(readFileSync(join(root, 'record.json'), 'utf8')), ['--prefix', 'skills', 'run', 'cli', '--', 'deploy', '--project', '/path with spaces', '--build', 'npm run build']);
    assert.equal(command('unknown').status, 2);
  } finally {
    rmSync(root, {recursive: true, force: true});
  }
});

test('complete commands include web checks and propagate web failure', () => {
  const root = mkdtempSync(join(tmpdir(), 'cellapp-web-command-'));
  try {
    mkdirSync(join(root, 'scripts'));
    copyFileSync(new URL('../scripts/project.mjs', import.meta.url), join(root, 'scripts/project.mjs'));
    mkdirSync(join(root, 'skills', '.git'), {recursive: true});
    writeFileSync(join(root, 'skills', 'package.json'), '{}');
    mkdirSync(join(root, 'node_modules', 'typescript', 'bin'), {recursive: true});
    writeFileSync(join(root, 'node_modules', 'typescript', 'bin', 'tsc'), '');
    mkdirSync(join(root, 'apps', 'web', 'dist'), {recursive: true});
    writeFileSync(join(root, 'apps', 'web', 'dist', 'index.html'), 'web artifact');
    mkdirSync(join(root, 'tests'));
    writeFileSync(join(root, 'tests', 'project.test.mjs'), '');
    mkdirSync(join(root, 'bin'));
    for (const tool of ['npm', 'go']) {
      writeFileSync(join(root, 'bin', tool), `#!${process.execPath}\nconst fs = require('node:fs'); const args = process.argv.slice(2); fs.appendFileSync(process.env.COMMAND_RECORD, JSON.stringify({tool: '${tool}', args}) + '\\n'); if (args.includes('@cellapp/web') && args.includes(process.env.FAIL_STAGE)) process.exit(29);\n`, {mode: 0o755});
    }
    const command = (action, failStage = '') => {
      writeFileSync(join(root, 'record'), '');
      const result = spawnSync(process.execPath, [join(root, 'scripts/project.mjs'), action], {
        encoding: 'utf8',
        env: {...process.env, PATH: `${join(root, 'bin')}:${process.env.PATH}`, COMMAND_RECORD: join(root, 'record'), FAIL_STAGE: failStage},
      });
      return {result, calls: readFileSync(join(root, 'record'), 'utf8').trim().split('\n').filter(Boolean).map((line) => JSON.parse(line))};
    };
    for (const action of ['typecheck', 'test', 'build']) {
      const {result, calls} = command(action);
      assert.equal(result.status, 0, result.stderr);
      assert.deepEqual(calls[0], {tool: 'npm', args: ['--prefix', 'skills', 'run', action]});
      const webCalls = calls.filter((call) => call.args.includes('@cellapp/web'));
      assert.deepEqual(webCalls.map((call) => call.args), action === 'test'
        ? [['run', 'build', '--workspace', '@cellapp/web'], ['run', 'lint', '--workspace', '@cellapp/web'], ['test', '--workspace', '@cellapp/web']]
        : [['run', action, '--workspace', '@cellapp/web']]);
      assert.equal(command(action, action).result.status, 29);
    }
    const lintFailure = command('test', 'lint');
    assert.equal(lintFailure.result.status, 29);
    assert.equal(lintFailure.calls.some((call) => call.tool === 'npm' && call.args[0] === 'test'), false);
  } finally {
    rmSync(root, {recursive: true, force: true});
  }
});

test('server public boundary check covers the server surface without rejecting the monorepo', () => {
  const result = spawnSync(process.execPath, ['scripts/check-public-boundary.mjs'], {encoding: 'utf8', cwd: join(import.meta.dirname, '..')});
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Public server source boundary passed/);
});

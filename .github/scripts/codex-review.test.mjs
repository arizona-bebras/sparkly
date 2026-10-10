import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';
import { codexArguments, codexEnvironment, formatReview, validateReview } from './codex-review.mjs';

test('Codex receives subscription auth location but not runner secrets or API credentials', () => {
  const environment = codexEnvironment({
    HOME: '/home/runner',
    CODEX_HOME: '/home/runner/.codex',
    PATH: '/usr/bin',
    GH_TOKEN: 'github-secret',
    OPENAI_API_KEY: 'api-secret',
    CODEX_API_KEY: 'api-secret',
    CODEX_ACCESS_TOKEN: 'override-secret',
    ACTIONS_RUNTIME_TOKEN: 'runner-secret',
    UNRELATED_SECRET: 'secret',
    GIT_CONFIG_VALUE_0: 'git-secret',
  });
  assert.deepEqual(environment, {
    HOME: '/home/runner',
    CODEX_HOME: '/home/runner/.codex',
    PATH: '/usr/bin',
  });
});

test('review disables execution and integrations while preserving saved auth', () => {
  const args = codexArguments('/tmp/review');
  assert.ok(args.includes('--ignore-user-config'));
  assert.ok(args.includes('--ignore-rules'));
  assert.ok(args.includes('--no-daemon'));
  assert.equal(args[args.indexOf('--sandbox') + 1], 'read-only');
  assert.equal(args[args.indexOf('--ask-for-approval') + 1], 'never');
  for (const feature of ['shell_tool', 'unified_exec', 'apps', 'plugins', 'hooks', 'multi_agent']) {
    assert.ok(args.some((value, index) => value === '--disable' && args[index + 1] === feature));
  }
  assert.ok(args.includes('web_search="disabled"'));
  assert.ok(!args.includes('--dangerously-bypass-approvals-and-sandbox'));
  assert.equal(args.at(-1), '-');
});

test('optional model is passed literally, without a shell', () => {
  const model = 'configured-model';
  const args = codexArguments('/tmp/review', model);
  assert.equal(args[args.indexOf('--model') + 1], model);
});

const review = {
  summary: 'Обнаружена ошибка.',
  limitations: [],
  findings: [
    { priority: 'P1', title: 'Ошибка', body: 'Объяснение', path: 'frontend/src/app.ts', line: 3 },
  ],
};

test('rejects findings outside changed files and invalid locations', () => {
  const paths = new Set(['frontend/src/app.ts']);
  assert.equal(validateReview(review, paths), review);
  assert.throws(() => validateReview(review, new Set()), /outside the PR/);
  assert.throws(() =>
    validateReview({ ...review, findings: [{ ...review.findings[0], line: 0 }] }, paths),
  );
  assert.throws(() =>
    validateReview({ ...review, findings: Array(21).fill(review.findings[0]) }, paths),
  );
});

test('report includes immutable source links and escapes model-supplied markup', () => {
  const markdown = formatReview(
    { ...review, summary: '<script>untrusted</script>' },
    'owner/repo',
    'a'.repeat(40),
  );
  assert.ok(!markdown.includes('<script>'));
  assert.match(markdown, /&lt;script&gt;/);
  assert.ok(markdown.includes(`/blob/${'a'.repeat(40)}/frontend/src/app.ts#L3`));
  assert.match(markdown, /тесты и команды из PR не выполнялись/);
});

test('reviews fetched PR data without checking out head or passing GitHub secrets to Codex', () => {
  const temporary = mkdtempSync(join(tmpdir(), 'codex-review-test-'));
  try {
    const repository = join(temporary, 'repository');
    const output = join(temporary, 'output');
    const binary = join(temporary, 'codex');
    mkdirSync(repository);
    const git = (...args) =>
      execFileSync('git', args, {
        cwd: repository,
        encoding: 'utf8',
        stdio: ['ignore', 'pipe', 'inherit'],
      }).trim();
    git('init');
    git('config', 'user.name', 'Review test');
    git('config', 'user.email', 'review@example.invalid');
    git('config', 'commit.gpgsign', 'false');
    git('config', 'core.hooksPath', '/dev/null');
    writeFileSync(join(repository, 'app.js'), 'export const value = 1;\n');
    git('add', '.');
    git('commit', '-m', 'base');
    const base = git('rev-parse', 'HEAD');
    writeFileSync(join(repository, 'app.js'), 'export const value = 2;\n');
    git('commit', '-am', 'head');
    const head = git('rev-parse', 'HEAD');
    git('update-ref', 'refs/pull/1/head', head);
    git('checkout', '--detach', base);
    git('remote', 'add', 'origin', repository);
    writeFileSync(
      binary,
      `#!${process.execPath}
const assert = require('node:assert/strict');
const fs = require('node:fs');
const args = process.argv.slice(2);
assert.equal(process.env.GH_TOKEN, undefined);
assert.equal(process.env.OPENAI_API_KEY, undefined);
if (args.includes('--version')) console.log('codex-cli mock');
else if (args.includes('--help')) console.log('--ignore-user-config --ignore-rules --ephemeral --output-schema');
else if (args[0] === 'login') console.log('Logged in using ChatGPT');
else {
  assert.ok(args.includes('--ignore-user-config'));
  const prompt = fs.readFileSync(0, 'utf8');
  assert.ok(prompt.includes('export const value = 2;'));
  assert.ok(!prompt.includes('github-secret'));
  fs.writeFileSync(args[args.indexOf('--output-last-message') + 1], JSON.stringify({
    summary: 'Проверено', limitations: [],
    findings: [{ priority: 'P2', title: 'Test finding', body: 'Test explanation', path: 'app.js', line: 1 }]
  }));
}
`,
      { mode: 0o755 },
    );
    execFileSync(
      process.execPath,
      [fileURLToPath(new URL('./codex-review.mjs', import.meta.url))],
      {
        cwd: repository,
        env: {
          ...process.env,
          CODEX_BIN: binary,
          GH_TOKEN: 'github-secret',
          OPENAI_API_KEY: 'api-secret',
          PR_BASE_SHA: base,
          PR_HEAD_SHA: head,
          PR_NUMBER: '1',
          REVIEW_OUTPUT_DIR: output,
          GITHUB_REPOSITORY: 'owner/repo',
          GITHUB_STEP_SUMMARY: join(temporary, 'summary.md'),
        },
        stdio: ['ignore', 'pipe', 'inherit'],
      },
    );
    assert.equal(git('rev-parse', 'HEAD'), base);
    assert.equal(readFileSync(join(repository, 'app.js'), 'utf8'), 'export const value = 1;\n');
    const check = JSON.parse(readFileSync(join(output, 'check.json'), 'utf8'));
    assert.equal(check.findings, 1);
    assert.ok(check.summary.includes(`/blob/${head}/app.js#L1`));
    assert.match(readFileSync(join(temporary, 'summary.md'), 'utf8'), /Проверено/);
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
});

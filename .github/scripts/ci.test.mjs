import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { makePlan, makeSummary } from './ci.mjs';

const projects = {
  frontend: { targets: { check: {}, build: {} } },
  backend: { targets: { check: {}, build: {} } },
  ml: { targets: { check: {} } },
  proto: { targets: { check: {} } },
};

test('runs affected projects and distinguishes check-only components', () => {
  const plan = makePlan(projects, ['ml'], ['ml/src/sparkly_ml/__main__.py']);
  assert.deepEqual(
    plan.filter((item) => item.affected).map((item) => item.project),
    ['ml'],
  );
  assert.equal(plan.find((item) => item.project === 'ml').build, false);
});

test('CI and shared tooling changes verify every component', () => {
  for (const file of ['.github/workflows/ci.yml', 'nx.json', 'pnpm-lock.yaml', '.nxignore']) {
    assert.ok(makePlan(projects, ['proto'], [file]).every((item) => item.affected));
  }
});

test('manual full runs and no affected projects are supported', () => {
  assert.ok(makePlan(projects, [], [], true).every((item) => item.affected));
  assert.ok(makePlan(projects, [], []).every((item) => !item.affected));
});

test('summary exposes missing results, failures, skipped builds and artifacts', () => {
  const plan = makePlan(projects, ['frontend', 'backend', 'ml'], []);
  const summary = makeSummary(plan, [
    {
      project: 'frontend',
      check: 'success',
      build: 'success',
      status: 'success',
      artifact: 'https://github.com/example/artifact/1',
    },
    { project: 'ml', check: 'failure', build: 'skipped', status: 'failure' },
  ]);
  assert.match(summary, /frontend.*✅ успешно.*\[скачать\]/);
  assert.match(summary, /backend.*❌ нет результата/);
  assert.match(summary, /ml.*❌ ошибка.*нет задачи/);
  assert.match(summary, /proto.*без изменений/);
});

test('overall CI gate handles empty matrices, missing reports and failed jobs', () => {
  const directory = mkdtempSync(join(tmpdir(), 'sparkly-ci-test-'));
  const script = fileURLToPath(new URL('./ci.mjs', import.meta.url));
  const run = (planned, componentResult, planResult = 'success') =>
    spawnSync(process.execPath, [script, 'summary'], {
      cwd: directory,
      encoding: 'utf8',
      env: {
        ...process.env,
        GITHUB_STEP_SUMMARY: join(directory, 'summary.md'),
        CI_PLAN: JSON.stringify(planned),
        PLAN_RESULT: planResult,
        COMPONENTS_RESULT: componentResult,
      },
    }).status;
  try {
    const empty = makePlan(projects, [], []);
    assert.equal(run(empty, 'skipped'), 0);
    assert.equal(run(empty, 'skipped', 'failure'), 1);
    const affected = makePlan(projects, ['backend'], []);
    assert.equal(run(affected, 'success'), 1);
    mkdirSync(join(directory, '.ci-results'));
    writeFileSync(
      join(directory, '.ci-results/backend.json'),
      JSON.stringify({
        project: 'backend',
        check: 'success',
        build: 'success',
        status: 'success',
      }),
    );
    assert.equal(run(affected, 'success'), 0);
    assert.equal(run(affected, 'failure'), 1);
    assert.equal(run(affected, 'cancelled'), 1);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

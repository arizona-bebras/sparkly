import { execFileSync } from 'node:child_process';
import { appendFileSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const components = {
  frontend: { runtime: 'node', artifact: 'frontend/build' },
  backend: { runtime: 'go', artifact: 'backend/bin/backend' },
  analytics: { runtime: 'go', artifact: 'analytics/bin/analytics' },
  agent: { runtime: 'rust', artifact: 'agent/target/ci-artifacts' },
  ml: { runtime: 'python', artifact: '' },
  proto: { runtime: 'node', artifact: '' },
  envoy: { runtime: 'node', artifact: '' },
  symbolicator: { runtime: 'rust', artifact: 'symbolicator/target/debug/sparkly-symbolicator' },
};

const nx = (...args) =>
  JSON.parse(
    execFileSync('pnpm', ['exec', 'nx', ...args, '--json'], {
      encoding: 'utf8',
      env: { ...process.env, NX_DAEMON: 'false', NX_TUI: 'false' },
    }),
  );

export function makePlan(projects, affected, changedFiles, forceAll = false) {
  const globalChange = changedFiles.some(
    (file) =>
      file.startsWith('.github/') ||
      file.startsWith('tools/') ||
      ['nx.json', '.nxignore', 'package.json', 'pnpm-workspace.yaml', 'pnpm-lock.yaml'].includes(
        file,
      ),
  );
  const selected = new Set(forceAll || globalChange ? Object.keys(projects) : affected);
  return Object.entries(projects)
    .sort(([a], [b]) => a.localeCompare(b))
    .flatMap(([project, config]) => {
      const check = Boolean(config.targets.check);
      const build = Boolean(config.targets.build);
      if (!check && !build) return [];
      if (!components[project]) throw new Error(`Add CI runtime metadata for project: ${project}`);
      return [{ project, ...components[project], check, build, affected: selected.has(project) }];
    });
}

function plan() {
  const all = nx('show', 'projects');
  const projects = Object.fromEntries(
    all.map((project) => [project, nx('show', 'project', project)]),
  );
  const changedFiles = execFileSync(
    'git',
    ['diff', '--name-only', process.env.NX_BASE, process.env.NX_HEAD],
    {
      encoding: 'utf8',
    },
  )
    .trim()
    .split('\n')
    .filter(Boolean);
  const planned = makePlan(
    projects,
    nx('show', 'projects', '--affected'),
    changedFiles,
    process.env.FORCE_ALL === 'true',
  );
  const include = planned.filter((component) => component.affected);
  const outputs = {
    matrix: JSON.stringify({ include: include.length ? include : [{ project: 'none' }] }),
    'has-work': String(include.length > 0),
    plan: JSON.stringify(planned),
  };
  if (process.env.GITHUB_OUTPUT) {
    for (const [key, value] of Object.entries(outputs))
      appendFileSync(process.env.GITHUB_OUTPUT, `${key}=${value}\n`);
  }
  const summary = [
    '## План CI',
    '',
    `Диапазон Nx: \`${process.env.NX_BASE}\` → \`${process.env.NX_HEAD}\`.`,
    '',
    '| Компонент | Проверки | Сборка | Запуск |',
    '| --- | --- | --- | --- |',
    ...planned.map(
      (item) =>
        `| ${item.project} | ${item.check ? 'check' : '—'} | ${item.build ? 'build' : '—'} | ${item.affected ? '▶️' : '⏭️ без изменений'} |`,
    ),
    '',
  ].join('\n');
  if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, summary);
  console.log(JSON.stringify(outputs, null, 2));
}

const status = (value) =>
  ({
    success: '✅ успешно',
    failure: '❌ ошибка',
    cancelled: '🚫 отменено',
    skipped: '⏭️ пропущено',
    missing: '❌ нет результата',
  })[value] ?? value;

function report() {
  const result = {
    project: process.env.PROJECT,
    check: process.env.CHECK_OUTCOME || 'skipped',
    build: process.env.BUILD_OUTCOME || 'skipped',
    status: process.env.JOB_STATUS,
    artifact: process.env.ARTIFACT_URL || '',
    upload: process.env.UPLOAD_OUTCOME || 'skipped',
    hasBuild: process.env.HAS_BUILD === 'true',
  };
  mkdirSync('.ci-results', { recursive: true });
  writeFileSync(join('.ci-results', `${result.project}.json`), JSON.stringify(result));
  const summary = `## ${result.project}\n\n| Проверки | Сборка | Результат |\n| --- | --- | --- |\n| ${status(result.check)} | ${result.hasBuild ? status(result.build) : '— нет задачи'} | ${status(result.status)} |\n\n${result.artifact ? `[Скачать сборку](${result.artifact})\n` : ''}`;
  if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, summary);
}

export function makeSummary(planned, results) {
  return [
    '## Результаты компонентов',
    '',
    '| Компонент | Проверки | Сборка | Итог | Артефакт |',
    '| --- | --- | --- | --- | --- |',
    ...planned.map((item) => {
      if (!item.affected) return `| ${item.project} | — | — | ⏭️ без изменений | — |`;
      const result = results.find((candidate) => candidate.project === item.project);
      if (!result) return `| ${item.project} | — | — | ${status('missing')} | — |`;
      return `| ${item.project} | ${item.check ? status(result.check) : '—'} | ${item.build ? status(result.build) : '— нет задачи'} | ${status(result.status)} | ${result.artifact ? `[скачать](${result.artifact})` : '—'} |`;
    }),
    '',
  ].join('\n');
}

function summary() {
  const planned = JSON.parse(process.env.CI_PLAN || '[]');
  let results = [];
  try {
    results = readdirSync('.ci-results')
      .filter((file) => file.endsWith('.json'))
      .map((file) => JSON.parse(readFileSync(join('.ci-results', file), 'utf8')));
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
  const markdown = makeSummary(planned, results);
  if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, markdown);
  console.log(markdown);
  if (
    process.env.PLAN_RESULT !== 'success' ||
    (planned.some((item) => item.affected) && process.env.COMPONENTS_RESULT !== 'success') ||
    planned.some(
      (item) =>
        item.affected &&
        !results.some((result) => result.project === item.project && result.status === 'success'),
    )
  ) {
    process.exitCode = 1;
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const command = { plan, report, summary }[process.argv[2]];
  if (!command) throw new Error('Usage: node .github/scripts/ci.mjs plan|report|summary');
  command();
}

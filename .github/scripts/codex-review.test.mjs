import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "node:test";
import {
  changedLineRanges,
  codexArguments,
  codexEnvironment,
  permissionArguments,
  runtimeReadPaths,
  reviewReport,
  validateReview,
} from "./codex-review.mjs";

const root = "/repository";
const review = {
  overall_correctness: "patch is incorrect",
  overall_explanation: "Ошибка в изменённом коде.",
  overall_confidence_score: 0.9,
  findings: [
    {
      title: "Ошибка",
      body: "Причина и последствия.",
      priority: 1,
      confidence_score: 0.95,
      code_location: { absolute_file_path: "/repository/app.js", line_range: { start: 3, end: 4 } },
    },
  ],
};
const changedFiles = new Map([["app.js", [{ start: 3, end: 5 }]]]);

test("extracts changed head-side ranges, excluding deleted-only hunks", () => {
  assert.deepEqual(changedLineRanges("@@ -1 +1 @@\n@@ -3,2 +3,3 @@\n@@ -9,2 +10,0 @@\n"), [
    { start: 1, end: 1 },
    { start: 3, end: 5 },
  ]);
});

test("validates official review fields, confidence, paths and ranges within changed lines", () => {
  assert.equal(validateReview(review, changedFiles, root), review);
  const invalidFinding = (overrides) => ({
    ...review,
    findings: [{ ...review.findings[0], ...overrides }],
  });
  for (const invalid of [
    { ...review, overall_confidence_score: 1.01 },
    { ...review, overall_correctness: "unknown" },
    invalidFinding({ confidence_score: -1 }),
    invalidFinding({ priority: 4 }),
    invalidFinding({
      code_location: { absolute_file_path: "/outside/app.js", line_range: { start: 3, end: 4 } },
    }),
    invalidFinding({
      code_location: { absolute_file_path: "/repository/app.js", line_range: { start: 2, end: 4 } },
    }),
    invalidFinding({
      code_location: { absolute_file_path: "/repository/app.js", line_range: { start: 4, end: 3 } },
    }),
    { ...review, findings: Array(21).fill(review.findings[0]) },
  ])
    assert.throws(() => validateReview(invalid, changedFiles, root));
});

test("renders verdict, confidence, immutable links and GitHub annotations", () => {
  const report = reviewReport(review, { root, repository: "owner/repo", sha: "a".repeat(40) });
  assert.equal(report.check.conclusion, "neutral");
  assert.match(report.markdown, /уверенность 90%/);
  assert.ok(report.markdown.includes(`/blob/${"a".repeat(40)}/app.js#L3-L4`));
  assert.deepEqual(report.check.annotations[0], {
    path: "app.js",
    start_line: 3,
    end_line: 4,
    annotation_level: "failure",
    title: "P1: Ошибка",
    message: "Причина и последствия.\n\nУверенность: 95%",
  });
  const clean = reviewReport(
    { ...review, overall_correctness: "patch is correct", findings: [] },
    { root, repository: "owner/repo", sha: "a".repeat(40) },
  );
  assert.equal(clean.check.conclusion, "success");
  assert.deepEqual(clean.check.annotations, []);
});

test("escapes model-supplied markdown and keeps check summary within API limit", () => {
  const report = reviewReport(
    { ...review, overall_explanation: "<script>[unsafe](url)</script>" },
    { root, repository: "owner/repo", sha: "a".repeat(40) },
  );
  assert.ok(!report.markdown.includes("<script>"));
  assert.match(report.markdown, /&lt;script&gt;/);
  const large = reviewReport(
    { ...review, findings: Array(20).fill({ ...review.findings[0], body: "Я".repeat(4000) }) },
    { root, repository: "owner/repo", sha: "a".repeat(40) },
  );
  assert.ok(Buffer.byteLength(large.check.summary) < 65_535);
});

test("Codex keeps saved auth locations but receives no API or runner tokens", () => {
  assert.deepEqual(
    codexEnvironment({
      HOME: "/home/runner",
      CODEX_HOME: "/home/runner/.codex",
      PATH: "/usr/bin",
      GH_TOKEN: "secret",
      OPENAI_API_KEY: "secret",
      CODEX_API_KEY: "secret",
      ACTIONS_RUNTIME_TOKEN: "secret",
      GIT_CONFIG_VALUE_0: "secret",
      ARBITRARY_SECRET: "secret",
    }),
    { HOME: "/home/runner", CODEX_HOME: "/home/runner/.codex", PATH: "/usr/bin" },
  );
  const args = codexArguments(
    "/tmp/review",
    "/tmp/review/repository",
    "model-with-spaces ; ignored",
  );
  assert.equal(args[args.indexOf("--model") + 1], "model-with-spaces ; ignored");
  assert.ok(args.includes("--ignore-user-config"));
  assert.ok(args.includes("--no-daemon"));
  assert.ok(args.includes("mcp_servers={}"));
  for (const feature of ["apps", "plugins", "hooks", "multi_agent", "shell_snapshot"]) {
    assert.ok(args.some((value, index) => value === "--disable" && args[index + 1] === feature));
  }
  for (const feature of ["shell_tool", "unified_exec"]) {
    assert.ok(args.some((value, index) => value === "--enable" && args[index + 1] === feature));
  }
  assert.equal(args[args.indexOf("--cd") + 1], "/tmp/review/repository");
  assert.ok(!args.includes("--sandbox"));
  assert.ok(args.includes('shell_environment_policy.inherit="none"'));
  assert.equal(args[args.indexOf("--output-schema") + 1], "/tmp/review/schema.json");
});

test("custom permissions allow snapshot reads while denying host reads, writes and networking", () => {
  const args = permissionArguments('/tmp/review/quote"in-path');
  assert.ok(args.includes('default_permissions="pr-review"'));
  const profile = args[1];
  assert.ok(profile.includes('":root"="deny"'));
  assert.ok(profile.includes('":minimal"="read"'));
  assert.ok(profile.includes(`${JSON.stringify('/tmp/review/quote"in-path')}="read"`));
  assert.ok(profile.includes('"/proc"="deny"'));
  assert.ok(profile.includes('"/run"="deny"'));
  assert.ok(profile.includes("network={enabled=false}"));
  assert.ok(!profile.includes('="write"'));
});

test("end-to-end: exposes the head snapshot and fails closed if sandbox isolation fails", () => {
  const temporary = mkdtempSync(join(tmpdir(), "codex-review-test-"));
  try {
    const repository = join(temporary, "repository");
    const output = join(temporary, "output");
    const binary = join(temporary, "codex");
    mkdirSync(repository);
    const git = (...args) =>
      execFileSync("git", args, {
        cwd: repository,
        encoding: "utf8",
        stdio: ["ignore", "pipe", "pipe"],
      }).trim();
    git("init");
    git("config", "user.name", "Review test");
    git("config", "user.email", "review@example.invalid");
    git("config", "commit.gpgsign", "false");
    git("config", "core.hooksPath", "/dev/null");
    writeFileSync(join(repository, "app.js"), "export const value = 1;\n");
    writeFileSync(join(repository, "dependency.js"), "export const limit = 10;\n");
    mkdirSync(join(repository, ".codex"));
    writeFileSync(join(repository, ".codex/config.toml"), 'approval_policy = "never"\n');
    symlinkSync("/etc/passwd", join(repository, "external-link"));
    symlinkSync("dependency.js", join(repository, "internal-link"));
    writeFileSync(join(repository, "asset.bin"), Buffer.from([0, 255, 128, 1]));
    git("add", ".");
    git("commit", "-m", "base");
    const base = git("rev-parse", "HEAD");
    writeFileSync(join(repository, "app.js"), "export const value = 2;\n");
    git("commit", "-am", "head");
    const head = git("rev-parse", "HEAD");
    git("update-ref", "refs/pull/1/head", head);
    git("checkout", "--detach", base);
    git("remote", "add", "origin", repository);
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
else if (args.includes('sandbox')) {
  assert.equal(args[args.indexOf('--permission-profile') + 1], 'pr-review');
  const root = args[args.indexOf('--cd') + 1];
  assert.equal(fs.readFileSync(root + '/app.js', 'utf8'), 'export const value = 2;\\n');
  assert.ok(args.some(arg => arg.includes('":root"="deny"')));
  if (fs.existsSync(__filename + '.fail-sandbox')) process.exit(21);
}
else {
  const prompt = fs.readFileSync(0, 'utf8');
  const source = JSON.parse(prompt.trim().split('\\n\\n').at(-1));
  assert.ok(prompt.includes('export const value = 2;'));
  assert.ok(!prompt.includes('github-secret'));
  assert.deepEqual(source.changedFiles[0].ranges, [{ start: 1, end: 1 }]);
  assert.equal(args[args.indexOf('--cd') + 1], source.repositoryRoot);
  assert.equal(fs.readFileSync(source.repositoryRoot + '/dependency.js', 'utf8'), 'export const limit = 10;\\n');
  assert.deepEqual(fs.readFileSync(source.repositoryRoot + '/asset.bin'), Buffer.from([0, 255, 128, 1]));
  assert.equal(fs.readlinkSync(source.repositoryRoot + '/internal-link'), 'dependency.js');
  assert.ok(!fs.existsSync(source.repositoryRoot + '/external-link'));
  assert.ok(!fs.existsSync(source.repositoryRoot + '/.codex/config.toml'));
  assert.ok(source.omittedSnapshotPaths.includes('.codex/config.toml'));
  fs.appendFileSync(__filename + '.exec-count', 'run\\n');
  const schema = JSON.parse(fs.readFileSync(args[args.indexOf('--output-schema') + 1], 'utf8'));
  assert.ok(schema.required.includes('overall_correctness'));
  fs.writeFileSync(args[args.indexOf('--output-last-message') + 1], JSON.stringify({
    overall_correctness: 'patch is incorrect', overall_explanation: 'Проверено', overall_confidence_score: 0.9,
    findings: [{ priority: 2, confidence_score: 0.8, title: 'Test finding', body: 'Test explanation',
      code_location: { absolute_file_path: source.changedFiles[0].absolute_file_path, line_range: { start: 1, end: 1 } } }]
  }));
}
`,
      { mode: 0o755 },
    );
    assert.deepEqual(runtimeReadPaths("codex", { PATH: temporary }), [binary]);
    execFileSync(
      process.execPath,
      [fileURLToPath(new URL("./codex-review.mjs", import.meta.url))],
      {
        cwd: repository,
        env: {
          ...process.env,
          CODEX_BIN: binary,
          GH_TOKEN: "github-secret",
          OPENAI_API_KEY: "api-secret",
          PR_BASE_SHA: base,
          PR_HEAD_SHA: head,
          PR_NUMBER: "1",
          REVIEW_OUTPUT_DIR: output,
          GITHUB_REPOSITORY: "owner/repo",
          GITHUB_STEP_SUMMARY: join(temporary, "summary.md"),
        },
        stdio: ["ignore", "pipe", "inherit"],
      },
    );
    assert.equal(git("rev-parse", "HEAD"), base);
    assert.equal(readFileSync(join(repository, "app.js"), "utf8"), "export const value = 1;\n");
    const check = JSON.parse(readFileSync(join(output, "check.json"), "utf8"));
    assert.equal(check.conclusion, "neutral");
    assert.equal(check.annotations[0].path, "app.js");
    assert.equal(check.annotations[0].start_line, 1);
    assert.match(readFileSync(join(temporary, "summary.md"), "utf8"), /Проверено/);
    writeFileSync(`${binary}.fail-sandbox`, "fail");
    assert.throws(
      () =>
        execFileSync(
          process.execPath,
          [fileURLToPath(new URL("./codex-review.mjs", import.meta.url))],
          {
            cwd: repository,
            env: {
              ...process.env,
              CODEX_BIN: binary,
              PR_BASE_SHA: base,
              PR_HEAD_SHA: head,
              PR_NUMBER: "1",
              REVIEW_OUTPUT_DIR: join(temporary, "failed-output"),
            },
            stdio: ["ignore", "pipe", "pipe"],
          },
        ),
      (error) => /sandbox preflight failed/.test(error.stderr.toString()),
    );
    assert.equal(readFileSync(`${binary}.exec-count`, "utf8"), "run\n");
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
});

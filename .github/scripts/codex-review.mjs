import { execFileSync, spawnSync } from "node:child_process";
import {
  appendFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  realpathSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { homedir } from "node:os";
import { dirname, isAbsolute, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const disabledFeatures = [
  "shell_snapshot",
  "code_mode_host",
  "apps",
  "plugins",
  "hooks",
  "multi_agent",
  "computer_use",
  "browser_use",
  "browser_use_external",
  "in_app_browser",
  "in_app_local_automation",
  "image_generation",
  "view_image",
  "skill_search",
  "memories",
];

export function codexEnvironment(environment) {
  const allowed = new Set([
    "PATH",
    "HOME",
    "USER",
    "LOGNAME",
    "SHELL",
    "TMPDIR",
    "TMP",
    "TEMP",
    "LANG",
    "LC_ALL",
    "CODEX_HOME",
    "XDG_CONFIG_HOME",
    "XDG_DATA_HOME",
    "XDG_CACHE_HOME",
    "DBUS_SESSION_BUS_ADDRESS",
    "SSL_CERT_FILE",
    "SSL_CERT_DIR",
    "NODE_EXTRA_CA_CERTS",
    "HTTP_PROXY",
    "HTTPS_PROXY",
    "ALL_PROXY",
    "NO_PROXY",
    "http_proxy",
    "https_proxy",
    "all_proxy",
    "no_proxy",
  ]);
  return Object.fromEntries(Object.entries(environment).filter(([name]) => allowed.has(name)));
}

export function runtimeReadPaths(binary, environment) {
  const executable =
    isAbsolute(binary) || binary.includes("/")
      ? binary
      : (environment.PATH || "")
          .split(":")
          .map((path) => join(path, binary))
          .find(existsSync);
  const real = realpathSync(executable);
  if (!real.endsWith(".js")) return [real];
  // npm launcher: permit only its installed package and the native companion package.
  const packageRoot = dirname(dirname(real));
  const companion = `codex-${process.platform}-${process.arch}`;
  return [
    packageRoot,
    join(dirname(packageRoot), companion),
    join(packageRoot, "node_modules/@openai", companion),
  ]
    .filter(existsSync)
    .map((path) => realpathSync(path));
}

export function permissionArguments(snapshot, runtime = []) {
  // Unlike the built-in read-only profile, this also denies reads outside the snapshot.
  const reads = [snapshot, ...runtime].map((path) => `${JSON.stringify(path)}="read"`).join(",");
  const profile = `permissions.pr-review={extends=":read-only",filesystem={":root"="deny",":minimal"="read",${reads},"/proc"="deny","/run"="deny"},network={enabled=false}}`;
  return ["-c", profile, "-c", 'default_permissions="pr-review"'];
}

export function codexArguments(directory, snapshot, model = "", runtime = []) {
  return [
    "--no-daemon",
    "--ask-for-approval",
    "never",
    "exec",
    "--skip-git-repo-check",
    "--ephemeral",
    "--ignore-user-config",
    "--ignore-rules",
    "--color",
    "never",
    "--cd",
    snapshot,
    ...permissionArguments(snapshot, runtime),
    "-c",
    'web_search="disabled"',
    "-c",
    "project_doc_max_bytes=0",
    "-c",
    "mcp_servers={}",
    "-c",
    'shell_environment_policy.inherit="none"',
    "-c",
    'shell_environment_policy.set={PATH="/usr/local/bin:/usr/bin:/bin",GIT_CONFIG_NOSYSTEM="1",GIT_CONFIG_GLOBAL="/dev/null",GIT_CONFIG_COUNT="2",GIT_CONFIG_KEY_0="core.hooksPath",GIT_CONFIG_VALUE_0="/dev/null",GIT_CONFIG_KEY_1="core.fsmonitor",GIT_CONFIG_VALUE_1="false"}',
    "--enable",
    "shell_tool",
    "--enable",
    "unified_exec",
    ...disabledFeatures.flatMap((feature) => ["--disable", feature]),
    ...(model ? ["--model", model] : []),
    "--output-schema",
    join(directory, "schema.json"),
    "--output-last-message",
    join(directory, "review.json"),
    "-",
  ];
}

export function materializeHead(git, head, snapshot) {
  mkdirSync(snapshot, { recursive: true });
  const omitted = [];
  const files = [];
  let totalBytes = 0;
  const entries = git(["ls-tree", "-rz", "--full-tree", head]).split("\0").filter(Boolean);
  for (const entry of entries) {
    const tab = entry.indexOf("\t");
    const [mode, type, oid] = entry.slice(0, tab).split(" ");
    const path = entry.slice(tab + 1);
    if (
      path.split("/").some((part) => [".codex", ".agents", ".git"].includes(part)) ||
      type !== "blob"
    ) {
      omitted.push(path);
      continue;
    }
    const target = resolve(snapshot, path);
    if (!target.startsWith(`${snapshot}/`)) throw new Error("Invalid snapshot path");
    const content = git(["cat-file", "blob", oid], undefined, null);
    totalBytes += content.length;
    if (totalBytes > 256 * 1024 * 1024) throw new Error("Repository snapshot exceeds 256 MB");
    mkdirSync(dirname(target), { recursive: true });
    if (mode === "120000") {
      const link = content.toString("utf8");
      if (!resolve(dirname(target), link).startsWith(`${snapshot}/`)) {
        omitted.push(path);
        continue;
      }
      symlinkSync(link, target);
    } else {
      // Git executable bits are intentionally not applied: this is a source snapshot.
      writeFileSync(target, content, { mode: 0o444 });
      files.push(path);
    }
  }
  return { omitted, files };
}

function verifySandbox(binary, snapshot, directory, files, environment, runtime) {
  if (!files.length) throw new Error("Repository snapshot has no readable files");
  const canary = join(directory, "outside-snapshot.txt");
  writeFileSync(canary, "sandbox read boundary probe");
  const args = [
    "--no-daemon",
    "sandbox",
    "--permission-profile",
    "pr-review",
    "--cd",
    snapshot,
    ...permissionArguments(snapshot, runtime),
    "--",
    "/bin/sh",
    "-c",
    'set -eu; cat "$1" >/dev/null; if cat "$2" >/dev/null 2>&1; then exit 21; fi; if ( : > "$3" ) 2>/dev/null; then exit 22; fi; if test -r "/proc/$4/status"; then exit 23; fi',
    "sandbox-probe",
    join(snapshot, files[0]),
    canary,
    join(snapshot, ".review-write-probe"),
    String(process.pid),
  ];
  const result = spawnSync(binary, args, {
    cwd: snapshot,
    env: environment,
    encoding: "utf8",
    timeout: 30_000,
  });
  if (result.status !== 0 || result.error) {
    throw new Error(
      `Codex sandbox preflight failed (exit ${result.status ?? result.error?.code}). Configure the Linux runner for Codex scoped filesystem permissions and unprivileged user namespaces. ${result.stderr || ""}`,
    );
  }
}

export function changedLineRanges(diff) {
  return [...diff.matchAll(/^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@/gm)]
    .map((match) => ({ start: Number(match[1]), count: Number(match[2] ?? 1) }))
    .filter(({ count }) => count > 0)
    .map(({ start, count }) => ({ start, end: start + count - 1 }));
}

function findingPath(finding, root) {
  const absolute = finding?.code_location?.absolute_file_path;
  if (typeof absolute !== "string" || !isAbsolute(absolute))
    throw new Error("Invalid absolute review path");
  const path = relative(root, absolute);
  if (path === ".." || path.startsWith("../") || isAbsolute(path))
    throw new Error("Review path outside repository");
  return path;
}

export function validateReview(review, changedFiles, root) {
  const text = (value, limit) =>
    typeof value === "string" && value.trim().length > 0 && value.length <= limit;
  const confidence = (value) =>
    typeof value === "number" && Number.isFinite(value) && value >= 0 && value <= 1;
  if (
    !review ||
    !["patch is correct", "patch is incorrect"].includes(review.overall_correctness) ||
    !text(review.overall_explanation, 6000) ||
    !confidence(review.overall_confidence_score) ||
    !Array.isArray(review.findings) ||
    review.findings.length > 20
  )
    throw new Error("Invalid structured review");
  for (const finding of review.findings) {
    const path = findingPath(finding, root);
    const range = finding.code_location.line_range;
    if (
      !text(finding.title, 80) ||
      !text(finding.body, 4000) ||
      !confidence(finding.confidence_score) ||
      !Number.isInteger(finding.priority) ||
      finding.priority < 0 ||
      finding.priority > 3 ||
      !Number.isInteger(range?.start) ||
      !Number.isInteger(range?.end) ||
      range.start < 1 ||
      range.end < range.start ||
      range.end - range.start >= 10 ||
      !changedFiles
        .get(path)
        ?.some((changed) => range.start >= changed.start && range.end <= changed.end)
    ) {
      throw new Error("Invalid finding or line range outside the PR diff");
    }
  }
  return review;
}

const escapeMarkdown = (value) =>
  value
    .replace(/[&<>]/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[character])
    .replace(/([\\`*_[\]#])/g, "\\$1");

export function reviewReport(review, { root, repository, sha, serverUrl = "https://github.com" }) {
  const verdict =
    review.overall_correctness === "patch is correct"
      ? "Ошибок не обнаружено"
      : "Обнаружены ошибки";
  const markdown = [
    "## Codex structured review",
    "",
    `Коммит: \`${sha}\`.`,
    "",
    `**${verdict}** · уверенность ${Math.round(review.overall_confidence_score * 100)}%`,
    "",
    escapeMarkdown(review.overall_explanation),
    "",
    ...review.findings.flatMap((finding) => {
      const path = findingPath(finding, root);
      const { start, end } = finding.code_location.line_range;
      const encoded = path.split("/").map(encodeURIComponent).join("/");
      return [
        `### P${finding.priority}: ${escapeMarkdown(finding.title)}`,
        "",
        `[${escapeMarkdown(path)}:${start}–${end}](${serverUrl}/${repository}/blob/${sha}/${encoded}#L${start}-L${end}) · уверенность ${Math.round(finding.confidence_score * 100)}%`,
        "",
        escapeMarkdown(finding.body),
        "",
      ];
    }),
    "Анализ diff и снимка head-коммита с чтением и поиском по репозиторию. Тесты и сборки не выполнялись. Замечания информационные.",
    "",
  ].join("\n");
  return {
    markdown,
    check: {
      title: `Codex: ${verdict.toLowerCase()} (${review.findings.length} замечаний)`,
      conclusion:
        review.findings.length || review.overall_correctness === "patch is incorrect"
          ? "neutral"
          : "success",
      summary:
        Buffer.byteLength(markdown) > 60_000
          ? `${Buffer.from(markdown).subarray(0, 60_000).toString("utf8")}\n\nПолный отчёт в артефакте workflow.`
          : markdown,
      annotations: review.findings.map((finding) => ({
        path: findingPath(finding, root),
        start_line: finding.code_location.line_range.start,
        end_line: finding.code_location.line_range.end,
        annotation_level:
          finding.priority <= 1 ? "failure" : finding.priority === 2 ? "warning" : "notice",
        title: `P${finding.priority}: ${finding.title}`,
        message: `${finding.body}\n\nУверенность: ${Math.round(finding.confidence_score * 100)}%`,
      })),
    },
  };
}

function main() {
  if (process.platform !== "linux")
    throw new Error("Repository review requires a Linux runner with Codex sandbox support");
  const directory = resolve(process.env.REVIEW_OUTPUT_DIR);
  const root = join(directory, "repository");
  const environment = codexEnvironment(process.env);
  const candidates = process.env.CODEX_BIN
    ? [process.env.CODEX_BIN]
    : ["codex", join(homedir(), ".local/bin/codex"), "/usr/local/bin/codex"];
  const binary = candidates.find(
    (candidate) =>
      spawnSync(candidate, ["--version"], { env: environment, encoding: "utf8", timeout: 10_000 })
        .status === 0,
  );
  if (!binary) throw new Error("Installed Codex CLI is unavailable to the runner user");
  const runtime = runtimeReadPaths(binary, environment);
  const help = execFileSync(binary, ["exec", "--help"], {
    env: environment,
    encoding: "utf8",
    timeout: 10_000,
  });
  for (const flag of ["--ignore-user-config", "--ignore-rules", "--ephemeral", "--output-schema"]) {
    if (!help.includes(flag)) throw new Error(`Update Codex CLI on the runner: missing ${flag}`);
  }
  const auth = spawnSync(binary, ["login", "status"], {
    env: environment,
    encoding: "utf8",
    timeout: 10_000,
  });
  if (auth.status !== 0 || !/chatgpt/i.test(`${auth.stdout}${auth.stderr}`))
    throw new Error("Run codex login with ChatGPT as the OS user running the runner service");
  const { PR_BASE_SHA: base, PR_HEAD_SHA: head, PR_NUMBER: number } = process.env;
  if (!/^[0-9a-f]{40}$/.test(base) || !/^[0-9a-f]{40}$/.test(head) || !/^\d+$/.test(number))
    throw new Error("Invalid PR metadata");
  const git = (args, env = process.env, encoding = "utf8") =>
    execFileSync(
      "git",
      [
        "--literal-pathspecs",
        "-c",
        "core.hooksPath=/dev/null",
        "-c",
        "core.fsmonitor=false",
        ...args,
      ],
      {
        env,
        encoding,
        maxBuffer: 256 * 1024 * 1024,
      },
    );
  const fetchEnvironment = { ...process.env };
  if (process.env.GH_TOKEN) {
    fetchEnvironment.GIT_CONFIG_COUNT = "1";
    fetchEnvironment.GIT_CONFIG_KEY_0 = `http.${process.env.GITHUB_SERVER_URL || "https://github.com"}/.extraheader`;
    fetchEnvironment.GIT_CONFIG_VALUE_0 = `AUTHORIZATION: basic ${Buffer.from(`x-access-token:${process.env.GH_TOKEN}`).toString("base64")}`;
  }
  // Review the immutable event commit, independently of GitHub's mutable PR refs.
  git(["fetch", "--no-tags", "origin", head], fetchEnvironment);
  const mergeBase = git(["merge-base", base, head]).trim();
  const diffArgs = ["diff", "--no-ext-diff", "--no-textconv", "--no-renames"];
  const paths = git([...diffArgs, "--name-only", "-z", mergeBase, head])
    .split("\0")
    .filter(Boolean);
  const diff = git([...diffArgs, "--unified=50", mergeBase, head]);
  if (Buffer.byteLength(diff) > 1_000_000) throw new Error("PR diff exceeds 1 MB; split the PR");
  const changedFiles = new Map(
    paths.map((path) => [
      path,
      changedLineRanges(git([...diffArgs, "--unified=0", mergeBase, head, "--", path])),
    ]),
  );
  let contextBytes = 0;
  const context = [];
  const omitted = [];
  for (const path of paths) {
    try {
      const content = git(["show", `${head}:${path}`]);
      const size = Buffer.byteLength(content);
      if (content.includes("\0") || size > 80_000 || contextBytes + size > 250_000)
        omitted.push(path);
      else {
        context.push({ path, content });
        contextBytes += size;
      }
    } catch {
      omitted.push(path);
    }
  }
  mkdirSync(directory, { recursive: true });
  const snapshot = materializeHead(git, head, root);
  verifySandbox(binary, root, directory, snapshot.files, environment, runtime);
  writeFileSync(
    join(directory, "schema.json"),
    readFileSync(new URL("../codex/review-schema.json", import.meta.url)),
  );
  const prompt = `${readFileSync(new URL("../codex/review-prompt.md", import.meta.url), "utf8")}\n\n${JSON.stringify(
    {
      base: mergeBase,
      head,
      repositoryRoot: root,
      diff,
      context,
      changedFiles: [...changedFiles].map(([path, ranges]) => ({
        path,
        absolute_file_path: join(root, path),
        ranges,
      })),
      omittedFullFileContext: omitted,
      omittedSnapshotPaths: snapshot.omitted,
    },
  )}`;
  const run = spawnSync(
    binary,
    codexArguments(directory, root, process.env.CODEX_REVIEW_MODEL || "", runtime),
    {
      cwd: root,
      env: environment,
      input: prompt,
      encoding: "utf8",
      timeout: 15 * 60 * 1000,
      maxBuffer: 16 * 1024 * 1024,
    },
  );
  if (run.status !== 0 || run.error)
    throw new Error(
      `Codex review failed (exit ${run.status ?? run.error?.code}); check runner login and subscription limits`,
    );
  const review = validateReview(
    JSON.parse(readFileSync(join(directory, "review.json"), "utf8")),
    changedFiles,
    root,
  );
  const report = reviewReport(review, {
    root,
    repository: process.env.GITHUB_REPOSITORY,
    sha: head,
    serverUrl: process.env.GITHUB_SERVER_URL,
  });
  writeFileSync(join(directory, "review.md"), report.markdown);
  writeFileSync(join(directory, "check.json"), JSON.stringify(report.check));
  if (process.env.GITHUB_STEP_SUMMARY)
    appendFileSync(process.env.GITHUB_STEP_SUMMARY, report.markdown);
  console.log(
    `Structured review completed: ${review.overall_correctness}, ${review.findings.length} findings`,
  );
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

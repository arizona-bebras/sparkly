import { spawnSync } from 'node:child_process';

// gofmt -l reports unformatted files but still exits successfully.
const result = spawnSync('gofmt', ['-l', '.'], { encoding: 'utf8' });
if (result.error) {
  console.error(result.error.message);
  process.exitCode = 1;
} else if (result.status !== 0) {
  process.stderr.write(result.stderr);
  process.exitCode = result.status ?? 1;
} else if (result.stdout.trim()) {
  console.error(`Run the project's format target to format:\n${result.stdout.trim()}`);
  process.exitCode = 1;
}

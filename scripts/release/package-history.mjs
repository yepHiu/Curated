import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

export function extractPreviousPackageHistoryCommit(markdown) {
  const lines = String(markdown)
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.startsWith("|"));

  const dataRows = lines.filter((line) => {
    if (!/[0-9a-fA-F]{7,40}\s*\/\s*/.test(line) && !/`[0-9a-fA-F]{7,40}`/.test(line)) {
      return false;
    }
    const normalized = line.replace(/\|/g, "").replace(/-/g, "").replace(/\s/g, "");
    return normalized.length > 0;
  });

  const lastRow = dataRows.at(-1);
  if (!lastRow) {
    return null;
  }

  const match = lastRow.match(/(?:`)?([0-9a-fA-F]{7,40})(?:`)?\s*\/\s*/);
  return match ? match[1] : null;
}

function normalizeGitLogOutput(raw) {
  return String(raw)
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .join("<br>");
}

export async function buildPackageHistoryChangeSummary({
  historyMarkdown,
  currentCommit,
  resolveCommit,
  loadGitLog,
}) {
  const previousCommit = extractPreviousPackageHistoryCommit(historyMarkdown);
  if (!previousCommit) {
    return "首条打包记录，无上一包可比对";
  }

  let previousResolved;
  try {
    previousResolved = await resolveCommit(previousCommit);
  } catch {
    return "无法解析上一条打包记录对应 commit";
  }

  let currentResolved;
  try {
    currentResolved = await resolveCommit(currentCommit);
  } catch {
    return "无法解析当前打包记录对应 commit";
  }

  if (!previousResolved) {
    return "无法解析上一条打包记录对应 commit";
  }

  if (!currentResolved) {
    return "无法解析当前打包记录对应 commit";
  }

  if (previousResolved === currentResolved) {
    return "无代码差异（同一提交重复打包）";
  }

  const rawLog = await loadGitLog(previousResolved, currentResolved);
  const normalized = normalizeGitLogOutput(rawLog);
  if (!normalized) {
    return "无代码差异（同一提交重复打包）";
  }

  return normalized;
}

function parseArgs(argv) {
  const [command, ...rest] = argv;
  const options = {};

  for (let index = 0; index < rest.length; index += 1) {
    const token = rest[index];
    if (!token.startsWith("--")) {
      throw new Error(`Unexpected argument: ${token}`);
    }
    const key = token.slice(2);
    const value = rest[index + 1];
    if (value == null || value.startsWith("--")) {
      throw new Error(`Missing value for --${key}`);
    }
    options[key] = value;
    index += 1;
  }

  return { command, options };
}

async function runCli() {
  const { command, options } = parseArgs(process.argv.slice(2));

  switch (command) {
    case "previous-commit": {
      const historyPath = options["history-path"];
      if (!historyPath) {
        throw new Error("Missing required --history-path argument.");
      }
      const historyMarkdown = await fs.readFile(path.resolve(historyPath), "utf8");
      const commit = extractPreviousPackageHistoryCommit(historyMarkdown);
      process.stdout.write(`${JSON.stringify({ commit })}\n`);
      return;
    }
    default:
      throw new Error(`Unsupported command: ${String(command)}`);
  }
}

const isDirectExecution = process.argv[1]
  ? path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))
  : false;

if (isDirectExecution) {
  runCli().catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}

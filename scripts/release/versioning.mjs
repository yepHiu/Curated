import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCHEMA_VERSION = 1;

function assertWholeNumber(value, label) {
  if (!Number.isInteger(value) || value < 0) {
    throw new Error(`Invalid ${label}: ${value}`);
  }
}

function normalizeVersionState(input) {
  if (input == null || typeof input !== "object" || Array.isArray(input)) {
    throw new Error("Release version file must be a JSON object.");
  }

  if (input.schema !== SCHEMA_VERSION) {
    throw new Error(
      `Unsupported release version schema: ${String(input.schema)}. Expected ${SCHEMA_VERSION}.`,
    );
  }

  const current = input.current;
  if (current == null || typeof current !== "object" || Array.isArray(current)) {
    throw new Error("Release version file is missing a valid current version.");
  }

  const major = Number(current.major);
  const minor = Number(current.minor);
  const patch = Number(current.patch);

  assertWholeNumber(major, "major version");
  assertWholeNumber(minor, "minor version");
  assertWholeNumber(patch, "patch version");

  return {
    schema: SCHEMA_VERSION,
    current: {
      major,
      minor,
      patch,
    },
  };
}

export function formatVersion(version) {
  return `${version.major}.${version.minor}.${version.patch}`;
}

export async function readVersionState(filePath) {
  const raw = await fs.readFile(filePath, "utf8");
  return normalizeVersionState(JSON.parse(raw));
}

async function writeVersionState(filePath, state) {
  const normalized = normalizeVersionState(state);
  const payload = `${JSON.stringify(normalized, null, 2)}\n`;
  await fs.mkdir(path.dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, payload, "utf8");
  return normalized;
}

export async function allocateNextPatchInFile(filePath) {
  const current = await readVersionState(filePath);
  const next = {
    schema: SCHEMA_VERSION,
    current: {
      major: current.current.major,
      minor: current.current.minor,
      patch: current.current.patch + 1,
    },
  };

  const saved = await writeVersionState(filePath, next);
  return {
    state: saved,
    version: formatVersion(saved.current),
  };
}

export async function setVersionBaseInFile(filePath, major, minor) {
  assertWholeNumber(major, "major version");
  assertWholeNumber(minor, "minor version");

  const next = {
    schema: SCHEMA_VERSION,
    current: {
      major,
      minor,
      patch: 0,
    },
  };

  const saved = await writeVersionState(filePath, next);
  return {
    state: saved,
    version: formatVersion(saved.current),
  };
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
  const filePath = options.file;

  if (!filePath) {
    throw new Error("Missing required --file argument.");
  }

  switch (command) {
    case "show": {
      const state = await readVersionState(filePath);
      process.stdout.write(`${JSON.stringify({
        state,
        version: formatVersion(state.current),
      })}\n`);
      return;
    }
    case "allocate": {
      const result = await allocateNextPatchInFile(filePath);
      process.stdout.write(`${JSON.stringify(result)}\n`);
      return;
    }
    case "set-base": {
      const major = Number(options.major);
      const minor = Number(options.minor);
      const result = await setVersionBaseInFile(filePath, major, minor);
      process.stdout.write(`${JSON.stringify(result)}\n`);
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

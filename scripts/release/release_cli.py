from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


# Allow invoking the CLI directly from the repo root without packaging it as a module first.
REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from scripts.release.release_lib.build_steps import (
    build_backend,
    build_frontend,
    migrate_history,
    package_installer,
    package_portable,
    publish_release,
    set_version_base,
    show_version,
    utc_build_stamp,
)


def main() -> None:
    parser = argparse.ArgumentParser(description="Curated release tooling")
    subparsers = parser.add_subparsers(dest="command", required=True)

    # Version inspection and manual base-line management.
    show_parser = subparsers.add_parser("show-version")
    show_parser.add_argument("--file", default="scripts/release/version.json")

    set_base_parser = subparsers.add_parser("set-version-base")
    set_base_parser.add_argument("--major", "--Major", dest="major", type=int, required=True)
    set_base_parser.add_argument("--minor", "--Minor", dest="minor", type=int, required=True)
    set_base_parser.add_argument("--file", default="scripts/release/version.json")

    # Low-level build steps used independently or as part of publish.
    frontend_parser = subparsers.add_parser("build-frontend")
    frontend_parser.add_argument("--version", "--Version", dest="version", required=True)
    frontend_parser.add_argument("--output-dir", default="release/frontend")

    backend_parser = subparsers.add_parser("build-backend")
    backend_parser.add_argument("--version", "--Version", dest="version", required=True)
    backend_parser.add_argument("--build-stamp", default=utc_build_stamp())
    backend_parser.add_argument("--output-dir", default="release/backend")
    backend_parser.add_argument("--binary-name", default="curated.exe")

    # Packaging commands share the same version source and history ledger defaults.
    portable_parser = subparsers.add_parser("package-portable")
    portable_parser.add_argument("--version", "--Version", dest="version")
    portable_parser.add_argument("--input-dir", default="release/Curated")
    portable_parser.add_argument("--output-dir", default="release/portable")
    portable_parser.add_argument("--version-file", default="scripts/release/version.json")
    portable_parser.add_argument("--history-path", default="docs/ops/package-build-history.csv")
    portable_parser.add_argument("--skip-history", action="store_true")

    installer_parser = subparsers.add_parser("package-installer")
    installer_parser.add_argument("--version", "--Version", dest="version")
    installer_parser.add_argument("--app-dir", default="release/Curated")
    installer_parser.add_argument("--output-dir", default="release/installer")
    installer_parser.add_argument("--template-path", default="scripts/release/windows/Curated.iss.tpl")
    installer_parser.add_argument("--version-file", default="scripts/release/version.json")
    installer_parser.add_argument("--history-path", default="docs/ops/package-build-history.csv")
    installer_parser.add_argument("--skip-history", action="store_true")

    publish_parser = subparsers.add_parser("publish")
    publish_parser.add_argument("--version", "--Version", dest="version")
    publish_parser.add_argument("--build-stamp", default=utc_build_stamp())
    publish_parser.add_argument("--output-dir", default="release")
    publish_parser.add_argument("--version-file", default="scripts/release/version.json")
    publish_parser.add_argument("--history-path", default="docs/ops/package-build-history.csv")

    # One-off migration from the legacy Markdown ledger to CSV.
    migrate_parser = subparsers.add_parser("migrate-history")
    migrate_parser.add_argument("--markdown-path", default="docs/ops/2026-04-02-package-build-history.md")
    migrate_parser.add_argument("--csv-path", default="docs/ops/package-build-history.csv")

    args = parser.parse_args()

    # Keep dispatch explicit so each subcommand stays easy to trace and debug.
    if args.command == "show-version":
        print(json.dumps(show_version(args.file), ensure_ascii=False))
        return

    if args.command == "set-version-base":
        result = set_version_base(args.major, args.minor, args.file)
        print("Release version base updated.")
        print(f"Version: {result['version']}")
        print(f"Source : {result['source']}")
        return

    if args.command == "build-frontend":
        build_frontend(args.version, args.output_dir)
        return

    if args.command == "build-backend":
        build_backend(args.version, args.build_stamp, args.output_dir, args.binary_name)
        return

    if args.command == "package-portable":
        package_portable(
            version=args.version,
            input_dir=args.input_dir,
            output_dir=args.output_dir,
            version_file=args.version_file,
            history_path=args.history_path,
            skip_history=args.skip_history,
        )
        return

    if args.command == "package-installer":
        package_installer(
            version=args.version,
            app_dir=args.app_dir,
            output_dir=args.output_dir,
            template_path=args.template_path,
            version_file=args.version_file,
            history_path=args.history_path,
            skip_history=args.skip_history,
        )
        return

    if args.command == "publish":
        publish_release(
            version=args.version,
            build_stamp=args.build_stamp,
            output_dir=args.output_dir,
            version_file=args.version_file,
            history_path=args.history_path,
        )
        return

    if args.command == "migrate-history":
        target = migrate_history(args.markdown_path, args.csv_path)
        print(target)
        return


if __name__ == "__main__":
    main()

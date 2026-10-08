"""Maintainer-only native engine preparation. End users only install Curated."""
import argparse
from pathlib import Path
import sys
ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))
from scripts.release.release_lib.macos_native import LOCK, prepare

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--lock', type=Path, default=ROOT / LOCK)
parser.add_argument('--output', type=Path, default=ROOT / '.workspace/native-player-macos-production')
parser.add_argument('--cache', type=Path, default=ROOT / '.workspace/native-player-macos-cache')
args = parser.parse_args()
prepare(args.lock.resolve(), args.output.resolve(), args.cache.resolve())

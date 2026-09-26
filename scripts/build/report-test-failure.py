"""Expose bounded test diagnostics as check annotations without changing test status."""
import pathlib
import re
import sys

text = pathlib.Path(sys.argv[1]).read_text(encoding='utf-8', errors='replace')
text = re.sub(r'\x1b\[[0-9;]*m', '', text)
lines = text.splitlines()
starts = [i for i, line in enumerate(lines) if line.startswith('--- FAIL:')]
blocks = ['\n'.join(lines[i:i + 16]) for i in starts]
if not blocks:
    blocks = ['\n'.join(lines[-65:])]
for block in blocks[:10]:
    message = block[:6000].replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A')
    print(f'::error title=Test failure details::{message}')

#!/usr/bin/env python3
# Copyright 2026 ALTESSA SOLUTIONS INC.
# Use of this source code is governed by an MIT-style license that can be
# found in the LICENSE file.
"""Check the complete externally defined symbol set, including private SQLite."""
import os
import pathlib
import re
import shlex
import subprocess
import sys
import tempfile

root = pathlib.Path(__file__).resolve().parent.parent
(root / 'build').mkdir(exist_ok=True)
expected = set(re.findall(r'^#define spd_\w+ (spdgo_\w+)$', (root / 'clib_symbols.h').read_text(), re.M))
actual = set()
with tempfile.TemporaryDirectory(dir=root / 'build') as tmp:
    for source in sorted(root.glob('clib_*.c')):
        obj = pathlib.Path(tmp) / (source.stem + '.o')
        cmd = shlex.split(os.environ.get('CC', 'cc'))
        cmd += ['-std=c2x', '-O0', '-I' + str(root / 'internal/clib/include'),
                '-I' + str(root / 'internal/clib/src'), '-I' + str(root / 'internal/clib/sqlite'),
                '-c', str(source), '-o', str(obj)]
        subprocess.run(cmd, check=True)
        flags = ['-g', '-U'] if sys.platform == 'darwin' else ['-g', '--defined-only']
        output = subprocess.check_output(['nm', *flags, str(obj)], text=True)
        for line in output.splitlines():
            fields = line.split()
            if len(fields) < 3:
                continue
            symbol = fields[-1]
            if sys.platform == 'darwin' and symbol.startswith('_'):
                symbol = symbol[1:]
            actual.add(symbol)
if actual != expected:
    raise SystemExit('external symbol mismatch: unexpected=' + repr(sorted(actual - expected))
                     + ' missing=' + repr(sorted(expected - actual)))
print(str(len(actual)) + ' prefixed snspd symbols; no externally defined SQLite symbols')

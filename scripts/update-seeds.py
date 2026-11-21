#!/usr/bin/env python3
# Copyright 2026 ALTESSA SOLUTIONS INC.
# Use of this source code is governed by an MIT-style license that can be
# found in the LICENSE file.
"""Regenerate synthetic fixtures only; never reads any user drawing."""
import hashlib
import pathlib
import shutil
import subprocess
import sys
import tempfile

root = pathlib.Path(__file__).resolve().parent.parent
source = pathlib.Path(sys.argv[1]).resolve()
revision = '0ac7ba43dcf4b798d9ac5c845dc2eee2b7d45b3b'
manifest = root / 'testdata/seeds.sha256'
expected = {pathlib.Path(line.split('  ', 1)[1]) for line in manifest.read_text().splitlines()}
fixtures = pathlib.Path('testdata/fixtures')
corpus = pathlib.Path('testdata/fuzz/FuzzOpen')
existing = {p.relative_to(root) for p in (root / fixtures).rglob('*.spd')}
existing.update(p.relative_to(root) for p in (root / corpus).rglob('*') if p.is_file())
if existing - expected:
    sys.exit(f'Unlisted fixture files: {sorted(map(str, existing - expected))}')

seeds = ['valid', 'empty', 'unknown-fields', 'empty-layers', 'huge-tid',
         'non-png', 'wal', 'unknown-version', 'malformed-protobuf', 'corrupt-page',
         'null-config', 'without-rowid']
with tempfile.TemporaryDirectory() as tmp:
    stage = pathlib.Path(tmp)
    for name in ('generate_fixtures.py', 'generate_btrees.py'):
        script = subprocess.check_output(['git', '-C', str(source), 'show', f'{revision}:tests/{name}'])
        (stage / name).write_bytes(script)
    (stage / fixtures).parent.mkdir()
    subprocess.run([sys.executable, '-B', str(stage / 'generate_fixtures.py'), str(stage / fixtures)],
                   check=True)
    # The upstream build also emits a completion marker beside the drawings.
    (stage / fixtures / 'stamp').unlink()
    (stage / corpus).mkdir(parents=True)
    for name in seeds:
        data = (stage / fixtures / (name + '.spd')).read_bytes()
        literal = ''.join('\\x%02x' % b for b in data)
        (stage / corpus / name).write_bytes(f'go test fuzz v1\n[]byte("{literal}")\n'.encode('ascii'))

    generated = {p.relative_to(stage) for p in (stage / 'testdata').rglob('*') if p.is_file()}
    if generated != expected:
        sys.exit(f'Generated files differ from manifest: {sorted(map(str, generated ^ expected))}')

    hashes = []
    for path in sorted(generated):
        (root / path).parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(stage / path, root / path)
        hashes.append(f'{hashlib.sha256((stage / path).read_bytes()).hexdigest()}  {path.as_posix()}\n')
    manifest.write_bytes(''.join(hashes).encode('ascii'))

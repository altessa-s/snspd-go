#!/usr/bin/env python3
# Copyright 2026 ALTESSA SOLUTIONS INC.
# Use of this source code is governed by an MIT-style license that can be
# found in the LICENSE file.
"""Offline transaction tests using synthetic sources and a read-only git stub."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("update-clib.sh")


class UpdateClibTest(unittest.TestCase):
    def run_update(self, operation="", destination="", extra_source=False):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            module, source, bin_dir = [base / p for p in ("module", "source", "bin")]
            for directory in (module / "scripts", source, bin_dir):
                directory.mkdir(parents=True)
            script = SCRIPT.read_text()
            if extra_source:
                script = script.replace("sources=(spd_log spd_model spd_parser)",
                                        "sources=(spd_log spd_model spd_parser spd_extra)")
            (module / "scripts/update-clib.sh").write_text(script)
            sources = [f"src/spd_{n}.c" for n in ("log", "model", "parser")]
            if extra_source:
                sources.append("src/spd_extra.c")
            files = sources + ["src/spd_internal.h", "LICENSE", "COPYRIGHT"]
            files += [f"include/snspd/{n}.h" for n in ("export", "log", "parser", "snspd", "types")]
            files += [f"vendor/sqlite/{n}" for n in ("sqlite3.c", "sqlite3.h", "VERSION", "README.md")]
            for name in files:
                path = source / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(f"synthetic {name}\n")
            originals = {name: f"old {name}\n" for name in (
                "internal/clib/old", "clib_digest.h", "internal/coexist/clib_digest.h",
                "clib_log.c", "clib_model.c", "clib_parser.c")}
            for name, text in originals.items():
                path = module / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(text)
            (bin_dir / "git").write_text('''#!/bin/bash
case "$*" in
  *status*) exit 0 ;;
  *describe*) echo synthetic ;;
  *rev-parse*) echo 0123456789012345678901234567890123456789 ;;
  *) exit 1 ;;
esac
''')
            for op in ("cp", "mv"):
                real = shutil.which(op)
                (bin_dir / op).write_text(f'''#!/bin/bash
if [[ "$FAIL_OP" == "{op}" && "${{!#}}" == *"$FAIL_DEST" ]]
then
    touch "$FAIL_MARKER"
    exit 73
fi
exec "{real}" "$@"
''')
            for path in bin_dir.iterdir():
                path.chmod(0o755)
            env = dict(os.environ, PATH=str(bin_dir) + os.pathsep + os.environ["PATH"],
                       FAIL_OP=operation, FAIL_DEST=destination, FAIL_MARKER=str(base / "failed"))
            result = subprocess.run(["bash", str(module / "scripts/update-clib.sh"), str(source)],
                                    env=env, text=True, capture_output=True, check=False)
            if operation:
                self.assertTrue((base / "failed").exists(), result.stderr)
                self.assertNotEqual(result.returncode, 0)
                actual = {str(p.relative_to(module)): p.read_text() for p in module.rglob("*")
                          if p.is_file() and p.parts[len(module.parts)] not in ("build", "scripts")}
                self.assertEqual(actual, originals, result.stderr)
            else:
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual((module / "clib_digest.h").read_bytes(),
                                 (module / "internal/coexist/clib_digest.h").read_bytes())
                self.assertFalse((module / "internal/clib/old").exists())
                if extra_source:
                    self.assertIn('#include "internal/clib/src/spd_extra.c"',
                                  (module / "clib_extra.c").read_text())

    def test_success(self):
        self.run_update()

    def test_added_source(self):
        self.run_update(extra_source=True)

    def test_added_source_rollback(self):
        # Fail after the new wrapper is installed: rollback must remove it.
        self.run_update("mv", "/module/internal/coexist/clib_digest.h", extra_source=True)

    def test_staging_failure(self):
        self.run_update("cp", "internal/coexist/clib_digest.h")

    def test_swap_failures(self):
        for name in ("clib_digest.h", "clib_log.c", "clib_model.c", "clib_parser.c",
                     "internal/coexist/clib_digest.h"):
            with self.subTest(name=name):
                self.run_update("mv", "/module/" + name)


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env bash
# Copyright 2026 ALTESSA SOLUTIONS INC.
# Use of this source code is governed by an MIT-style license that can be
# found in the LICENSE file.

# Re-vendor snspd-lib into internal/clib: the library sources and headers,
# its licence, and the tag they were taken at. The tests and tools of the C
# library stay behind; this module tests through its own API.
#
#   scripts/update-clib.sh ../snspd-lib
#
# The new tree is assembled and checked beside the old one and swapped in
# whole: on any failure the previous tree and digest stay as they were.
# clib_digest.h in the package root is rewritten with a digest of the vendored
# files, because Go's build cache tracks the package directory and not
# internal/clib; the digest is what makes a change down there rebuild the
# package.

set -euo pipefail

allow_dirty=false
if [[ ${1:-} == --allow-dirty ]]
then
    allow_dirty=true
    shift
fi
if [[ $# -ne 1 ]]
then
    echo "usage: $0 [--allow-dirty] <path-to-snspd-lib>" >&2
    exit 2
fi
if ! source_dir=$(cd "$1" && pwd -P)
then
    echo "library directory does not exist or is inaccessible: $1" >&2
    exit 1
fi
if ! module_dir=$(cd "$(dirname "$0")/.." && pwd -P)
then
    echo "cannot resolve module directory" >&2
    exit 1
fi
if [[ ! -f "$source_dir/include/snspd/snspd.h" ]]
then
    echo "not a snspd-lib directory: $source_dir (missing include/snspd/snspd.h)" >&2
    exit 1
fi
internal="$module_dir/internal"
clib="$internal/clib"

# Never copy from a tree that would be moved away during replacement.
case "$source_dir/" in "$clib/"*)
    echo "refusing source inside vendored destination: $source_dir" >&2
    exit 1
    ;;
esac
case "$module_dir/" in "$source_dir/"*)
    echo "refusing source containing module: $source_dir" >&2
    exit 1
    ;;
esac
if ! state=$(GIT_OPTIONAL_LOCKS=0 git --no-optional-locks -C "$source_dir" status --porcelain)
then
    echo "not a readable library checkout: $source_dir" >&2
    exit 1
fi
if [[ -n "$state" && $allow_dirty == false ]]
then
    echo "refusing dirty library checkout (use --allow-dirty explicitly): $source_dir" >&2
    exit 1
fi
version=$(GIT_OPTIONAL_LOCKS=0 git --no-optional-locks -C "$source_dir" describe --tags --always)
if [[ -n "$state" ]]
then
    version="$version-dirty"
fi

if command -v sha256sum > /dev/null
then
    sha256() { sha256sum | cut -d' ' -f1
    }
elif command -v shasum > /dev/null
then
    sha256() { shasum -a 256 | cut -d' ' -f1
    }
else
    echo "neither sha256sum nor shasum is installed" >&2
    exit 1
fi

sources=(spd_log spd_model spd_parser)
headers=(export log parser snspd types)

mkdir -p "$internal" "$module_dir/build"
staging=$(mktemp -d "$module_dir/build/clib-staging.XXXXXX")
retired="$module_dir/build/clib-retired.$$"
digest_tmp="$module_dir/build/clib_digest.h.$$"
generated=$(mktemp -d "$module_dir/build/clib-generated.XXXXXX")
backups=$(mktemp -d "$module_dir/build/clib-backups.XXXXXX")
replaced=()
tree_installed=false
finished=false

cleanup() {
    local status=$?
    # Stopped anywhere in the swap: the previous tree goes back in place.
    if [[ $finished == false ]]
    then
      for file in ${replaced[@]+"${replaced[@]}"}
      do
        if [[ -f "$backups/$file" ]]
        then
            mv "$backups/$file" "$module_dir/$file"
        else
            rm -f "$module_dir/$file"
        fi
      done
      if [[ $tree_installed == true ]]
      then
          rm -rf "$clib"
      fi
      if [[ -d "$retired" ]]
      then
        if [[ -e "$clib" ]]
        then
            rm -rf "$clib"
        fi
        mv "$retired" "$clib"
      fi
    fi
    rm -rf "$staging" "$digest_tmp" "$generated" "$backups"
    exit "$status"
}
trap cleanup EXIT

# SQLite goes to sqlite/, not vendor/sqlite/: the Go module zip leaves out
# every directory named vendor, so consumers would get no sqlite3.c.
mkdir -p "$staging/src" "$staging/include/snspd" "$staging/sqlite"
for name in "${sources[@]}"
do
    cp "$source_dir/src/$name.c" "$staging/src/"
done
cp "$source_dir/src/spd_internal.h" "$staging/src/"
for name in "${headers[@]}"
do
    cp "$source_dir/include/snspd/$name.h" "$staging/include/snspd/"
done
cp "$source_dir/LICENSE" "$source_dir/COPYRIGHT" "$staging/"
printf '%s\n' "$version" > "$staging/VERSION"
GIT_OPTIONAL_LOCKS=0 git --no-optional-locks -C "$source_dir" rev-parse HEAD > "$staging/COMMIT"
cp "$source_dir"/vendor/sqlite/{sqlite3.c,sqlite3.h,VERSION,README.md} "$staging/sqlite/"

# Every file this module compiles or ships must be there and not empty; a
# source the library gains or loses means the wrappers change first.
expected=(VERSION COMMIT LICENSE COPYRIGHT src/spd_internal.h sqlite/sqlite3.c sqlite/sqlite3.h sqlite/VERSION sqlite/README.md)
for name in "${sources[@]}"
do
    expected+=("src/$name.c")
done
for name in "${headers[@]}"
do
    expected+=("include/snspd/$name.h")
done
for file in "${expected[@]}"
do
    if [[ ! -s "$staging/$file" ]]
    then
        echo "missing or empty: $file" >&2
        exit 1
    fi
done
for file in "$source_dir"/src/*.c
do
    name=$(basename "$file" .c)
    if [[ ! -f "$staging/src/$name.c" ]]
    then
        echo "$name.c is not listed in sources" >&2
        exit 1
    fi
done

for file in "$source_dir"/include/snspd/*.h
do
    name=$(basename "$file")
    if [[ ! -f "$staging/include/snspd/$name" ]]
    then
        echo "$name is not listed in headers" >&2
        exit 1
    fi
done

version=$(cat "$staging/VERSION")
digest=$(
cd "$staging"
find . -type f | LC_ALL=C sort | while read -r file
do
    printf '%s %s\n' "$(sha256 < "$file")" "$file"
done | sha256)

cat > "$digest_tmp" <<EOF
// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// Generated by scripts/update-clib.sh. Do not edit.
//
// The vendored snspd-lib, named by its tag and by a digest of every file
// under internal/clib. Each clib_*.c includes this file, so a change to the
// vendored sources changes a file in the package directory, which is what
// Go's build cache tracks.

#ifndef SNSPD_GO_CLIB_DIGEST_H
#define SNSPD_GO_CLIB_DIGEST_H

#define SNSPD_GO_CLIB_VERSION "$version"
#define SNSPD_GO_CLIB_DIGEST "$digest"

#endif
EOF

# SQLite is included exactly once by spd_parser.c with SQLITE_API/EXTERN static.
# Do not create a second sqlite3.c wrapper.
for name in "${sources[@]}"
do
    cat > "$generated/clib_${name#spd_}.c" <<EOF
// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.
// Generated by scripts/update-clib.sh. Do not edit.
#include "clib_digest.h"
#include "clib_symbols.h"
#include "clib_sqlite.h"
#include "internal/clib/src/$name.c"
EOF
done

# Stage every generated file and save its old contents before changing the tree.
files=(clib_digest.h)
for name in "${sources[@]}"
do
    files+=("clib_${name#spd_}.c")
done
mv "$digest_tmp" "$generated/clib_digest.h"
if [[ -d "$internal/coexist" ]]
then
    files+=(internal/coexist/clib_digest.h)
    mkdir -p "$generated/internal/coexist"
    cp "$generated/clib_digest.h" "$generated/internal/coexist/clib_digest.h"
fi
for file in "${files[@]}"
do
    if [[ -e "$module_dir/$file" ]]
    then
        mkdir -p "$backups/$(dirname "$file")"
        cp "$module_dir/$file" "$backups/$file"
    fi
done

if [[ -d "$clib" ]]
then
    mv "$clib" "$retired"
fi
mv "$staging" "$clib"
tree_installed=true
for file in "${files[@]}"
do
    mv "$generated/$file" "$module_dir/$file"
    replaced+=("$file")
done
finished=true
rm -rf "$retired"

echo "vendored snspd-lib $version ($digest)"

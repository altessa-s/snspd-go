// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.
// Match snspd-lib's SQLite build; the amalgamation remains inside spd_parser.c.
#ifndef SNSPD_GO_SQLITE_H
#define SNSPD_GO_SQLITE_H
#define SQLITE_OS_OTHER 1
#define SQLITE_THREADSAFE 0
#define SQLITE_DEFAULT_MEMSTATUS 0
#define SQLITE_ZERO_MALLOC 1
#define SQLITE_OMIT_LOAD_EXTENSION 1
#define SQLITE_DQS 0
#define SQLITE_OMIT_WAL 1
#define SQLITE_OMIT_SHARED_CACHE 1
#define SQLITE_OMIT_UTF16
#define SQLITE_OMIT_DEPRECATED 1
#define SQLITE_OMIT_TCL_VARIABLE 1
#define SQLITE_OMIT_TRACE 1
#define SQLITE_OMIT_LOCALTIME 1
#define SQLITE_TEMP_STORE 3
#define SQLITE_DEFAULT_LOOKASIDE 0,0
#define SQLITE_ENABLE_API_ARMOR 1
#define SQLITE_MAX_LENGTH 67108864
#define SQLITE_MAX_SQL_LENGTH 4096
#define SQLITE_MAX_COLUMN 32
#define SQLITE_MAX_EXPR_DEPTH 32
#define SQLITE_MAX_VARIABLE_NUMBER 8
#endif

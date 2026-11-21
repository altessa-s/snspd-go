// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.
// Prefix every external snspd symbol, including private archive helpers.
#ifndef SNSPD_GO_CLIB_SYMBOLS_H
#define SNSPD_GO_CLIB_SYMBOLS_H
#define spd__internal_alloc spdgo_spd__internal_alloc
#define spd__internal_charge spdgo_spd__internal_charge
#define spd__internal_dealloc spdgo_spd__internal_dealloc
#define spd__internal_keep spdgo_spd__internal_keep
#define spd__internal_log_write spdgo_spd__internal_log_write
#define spd_config_find spdgo_spd_config_find
#define spd_drawing_free spdgo_spd_drawing_free
#define spd_log_install spdgo_spd_log_install
#define spd_log_silence spdgo_spd_log_silence
#define spd_open spdgo_spd_open
#define spd_options_default spdgo_spd_options_default
#define spd_painted_bounds spdgo_spd_painted_bounds
#define spd_status_string spdgo_spd_status_string
#define spd_tile_rect spdgo_spd_tile_rect
#endif

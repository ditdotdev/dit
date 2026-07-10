#!/bin/bash
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

/bin/bash -c printenv > /usr/share/nginx/html/index.html
nginx -g "daemon off;"
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Fixture for image-pull-policy.bats: an image that exists only in the local
# docker cache. The tag it is built under (ditdotdev/pull-policy-local:test)
# is never pushed, so any docker pull of it fails deterministically.
FROM ditdotdev/nginx-test:latest

LABEL dev.dit.test="image-pull-policy"

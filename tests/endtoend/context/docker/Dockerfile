# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM nginx:alpine

RUN apk update
RUN apk upgrade
RUN apk add bash

VOLUME /data/test

ADD start.sh /
RUN chmod +x /start.sh
CMD ["/start.sh"]
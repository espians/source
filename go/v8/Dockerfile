# Public Domain (-) 2018-present, The Espian Source Authors.
# See the Espian Source UNLICENSE file for details.

# Using ubuntu:16.04
FROM ubuntu@sha256:e348fbbea0e0a0e73ab0370de151e7800684445c509d46195aef73e090a49bd6

RUN apt-get update
RUN apt-get install -y build-essential curl git lsb-release locales \
    python-dev subversion sudo

RUN echo ttf-mscorefonts-installer msttcorefonts/accepted-mscorefonts-eula select true | debconf-set-selections

COPY v8.sh v8.sh
COPY version.sh version.sh

CMD ["/v8.sh"]

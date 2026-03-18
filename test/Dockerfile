# hfc-test:latest
FROM hfc-base:latest

RUN apt-get update && apt-get install -y \
    libtool && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /root/hfc/test

RUN bash build_freetype2.sh

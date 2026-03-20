# hfc-test:latest
FROM hfc-base:latest

RUN apt-get update && apt-get install -y \
    libtool file zlib1g-dev libarchive-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /root/hfc/

RUN cd test/ && \
    bash build_freetype2.sh && \
    bash build_sqlite3.sh && \
    bash build_libpng.sh && \
    bash build_harfbuzz.sh

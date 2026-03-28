# hfc-test:latest
FROM hfc-base:latest

RUN apt-get update && apt-get install -y \
    libtool file zlib1g-dev libarchive-dev vim sqlite3 libsqlite3-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /root/hfc/

RUN git submodule update --init --recursive && cd test/ && \
    bash build_freetype2.sh && \
    bash build_sqlite3.sh && \
    bash build_libpng.sh && \
    bash build_harfbuzz.sh


WORKDIR /opt
RUN wget https://github.com/grubwithu/hfc-introspector/releases/download/Alpha/targets.tgz && \
    tar -xzf targets.tgz && rm targets.tgz && \
    cd targets/ && bash hfc.sh && cd .. && \
    cd fuzzers/ && bash hfc.sh

WORKDIR /root/hfc/

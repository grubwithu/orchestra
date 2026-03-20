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


WORKDIR /opt
RUN wget https://github.com/grubwithu/hfc-introspector/releases/download/Alpha/targets.tgz && \
    tar -xzf targets.tgz && rm targets.tgz && \
    pushd targets && bash hfc.sh && popd && \
    cd fuzzers && bash hfc.sh

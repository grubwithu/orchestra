# hfc-test:latest
FROM hfc-base:latest

RUN apt-get update && apt-get install -y \
    libtool file zlib1g-dev libarchive-dev vim sqlite3 libsqlite3-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /root/hfc/

RUN git submodule update --init --recursive && cd test/ && \
    bash build_bloaty.sh && \
    bash build_freetype2.sh && \
    bash build_harfbuzz.sh && \
    bash build_libjpeg.sh && \
    bash build_libpng.sh && \
    bash build_libxml2.sh && \
    bash build_proj4.sh && \
    bash build_sqlite3.sh


WORKDIR /opt
RUN wget https://github.com/grubwithu/pfuzzer/releases/download/Alpha0.1/targets.tgz && \
    tar -xzf targets.tgz && rm targets.tgz && \
    cd targets/ && bash hfc.sh && cd .. && \
    cd fuzzers/ && bash hfc.sh

RUN wget https://github.com/grubwithu/pfuzzer/releases/download/Alpha0.1/seeds.tgz && \
    tar -xzf seeds.tgz && rm seeds.tgz && \
    mv seeds/* /root/hfc/test/

WORKDIR /root/hfc/

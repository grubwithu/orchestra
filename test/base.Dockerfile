# hfc-test:latest

# To facilitate deployment on the cluster when running batch experiments,
# we built all the programs under test into the same image,
# which may result in a very large image...

FROM hfc-base:latest

RUN apt-get update && apt-get install -y \
    libtool file zlib1g-dev libarchive-dev vim \
    sqlite3 libsqlite3-dev pkg-config libfl-dev \
    libpsl-dev libidn2-dev libnghttp2-dev gperf \
    meson ninja-build flex bison python3-venv && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /opt
RUN wget https://github.com/grubwithu/pfuzzer/releases/download/Alpha0.3/targets.tgz && \
    tar -xzf targets.tgz && rm targets.tgz && \
    wget https://github.com/grubwithu/pfuzzer/releases/download/Alpha0.3/seeds.tgz && \
    tar -xzf seeds.tgz && rm seeds.tgz

COPY script/prepare_hfc_artifacts.sh /usr/local/bin/prepare_hfc_artifacts.sh
RUN chmod +x /usr/local/bin/prepare_hfc_artifacts.sh



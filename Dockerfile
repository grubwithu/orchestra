# hfc-base:latest
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    wget git cmake curl \
    gnupg lsb-release \
    software-properties-common \
    build-essential \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://apt.llvm.org/llvm.sh && \
    chmod +x llvm.sh && \
    ./llvm.sh 18 && \
    apt-get update && apt-get install -y \
    clang-18 llvm-18 lld-18 \
    && rm llvm.sh \
    && rm -rf /var/lib/apt/lists/*

RUN update-alternatives --install /usr/bin/clang clang /usr/bin/clang-18 100 && \
    update-alternatives --install /usr/bin/clang++ clang++ /usr/bin/clang++-18 100 && \
    update-alternatives --install /usr/bin/llvm-config llvm-config /usr/bin/llvm-config-18 100 && \
    update-alternatives --install /usr/bin/lld lld /usr/bin/lld-18 100 && \
    update-alternatives --install /usr/bin/opt opt /usr/bin/opt-18 100 && \
    update-alternatives --install /usr/bin/llvm-link llvm-link /usr/bin/llvm-link-18 100 && \
    update-alternatives --install /usr/bin/llvm-cov llvm-cov /usr/bin/llvm-cov-18 100 && \
    update-alternatives --install /usr/bin/llvm-profdata llvm-profdata /usr/bin/llvm-profdata-18 100

ENV GO_VERSION=1.26.1
RUN ARCH="$(dpkg --print-architecture)" && \
    case "${ARCH}" in \
        amd64) GO_ARCH="amd64" ;; \
        arm64) GO_ARCH="arm64" ;; \
        *) echo "Unsupported architecture: ${ARCH}" && exit 1 ;; \
    esac && \
    wget "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" && \
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" && \
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

RUN go install github.com/SRI-CSL/gllvm/cmd/...@latest

WORKDIR /opt

RUN wget https://github.com/grubwithu/pfuzzer/releases/download/Alpha0.1/fuzzers.tgz && \
    tar -zxf fuzzers.tgz && rm fuzzers.tgz && cd fuzzers && bash build.sh

ENV AFL_SYSTEM_CONFIG="/opt/fuzzers/AFLplusplus/afl-system-config"

ADD "https://api.github.com/repos/grubwithu/hfc-introspector/commits?per_page=1" /dev/null
RUN git clone https://github.com/grubwithu/hfc-introspector.git && \
    cd hfc-introspector && mkdir build && cd build && \
    CC=clang CXX=clang++ cmake .. && make
ENV FUZZ_INTRO="/opt/hfc-introspector/build/FuzzIntrospector.so" 

WORKDIR /root

ADD "https://api.github.com/repos/grubwithu/hfc/commits?per_page=1" /dev/null

RUN git clone --depth 1 https://github.com/grubwithu/orchestra.git && \
    ln -s /root/orchestra /root/hfc && cd orchestra && \
    git submodule update --init --recursive pfuzzer && \
    make && cd pfuzzer && bash build.sh

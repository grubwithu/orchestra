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
    ./llvm.sh 21 && \
    apt-get update && apt-get install -y \
    clang-21 \
    llvm-21 \
    lld-21 \
    && rm llvm.sh \
    && rm -rf /var/lib/apt/lists/*

RUN update-alternatives --install /usr/bin/clang clang /usr/bin/clang-21 100 && \
    update-alternatives --install /usr/bin/clang++ clang++ /usr/bin/clang++-21 100 && \
    update-alternatives --install /usr/bin/llvm-config llvm-config /usr/bin/llvm-config-21 100 && \
    update-alternatives --install /usr/bin/lld lld /usr/bin/lld-21 100 && \
    update-alternatives --install /usr/bin/opt opt /usr/bin/opt-21 100 && \
    update-alternatives --install /usr/bin/llvm-link llvm-link /usr/bin/llvm-link-21 100

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

WORKDIR /opt
RUN git clone https://github.com/grubwithu/hfc-introspector.git && \
    cd hfc-introspector && mkdir build && cd build && \
    CC=clang-21 CXX=clang++-21 cmake .. && make
ENV FUZZ_INTRO="/opt/hfc-introspector/build/FuzzIntrospector.so" 

RUN go install github.com/SRI-CSL/gllvm/cmd/...@latest

RUN wget https://github.com/grubwithu/hfc-introspector/releases/download/Alpha/fuzzers.tgz && \
    wget https://github.com/grubwithu/hfc-introspector/releases/download/Alpha/targets.tgz && \
    tar -xzf fuzzers.tgz && tar -xzf targets.tgz && rm fuzzers.tgz targets.tgz && \
    cd fuzzers && bash build.sh

WORKDIR /root

ADD "https://api.github.com/repos/grubwithu/hfc/commits?per_page=1" /dev/null

RUN git clone https://github.com/grubwithu/hfc.git && \
    cd hfc && git submodule update --init --recursive && make && \
    cd pfuzzer && bash build.sh

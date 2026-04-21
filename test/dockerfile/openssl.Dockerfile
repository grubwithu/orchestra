FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/openssl && \
    cd submodule/ && bash ../script/build_openssl.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

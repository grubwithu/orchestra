FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/mbedtls && \
    cd submodule/ && bash ../script/build_mbedtls.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

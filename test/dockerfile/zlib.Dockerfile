FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/zlib && \
    cd submodule/ && bash ../script/build_zlib.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

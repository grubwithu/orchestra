FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/libpcap && \
    cd submodule/ && bash ../script/build_libpcap.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

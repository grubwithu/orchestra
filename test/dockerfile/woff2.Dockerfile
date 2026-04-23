FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/woff2 && \
    cd submodule/ && bash ../script/build_woff2.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

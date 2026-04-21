FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/libxml2 && \
    cd submodule/ && bash ../script/build_libxml2.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

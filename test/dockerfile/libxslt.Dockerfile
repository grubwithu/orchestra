FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/libxslt && \
    cd submodule/ && bash ../script/build_libxslt.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

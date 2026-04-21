FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/openthread && \
    cd submodule/ && bash ../script/build_openthread.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

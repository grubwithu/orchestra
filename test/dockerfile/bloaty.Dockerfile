FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/bloaty && \
    cd submodule/ && bash ../script/build_bloaty.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

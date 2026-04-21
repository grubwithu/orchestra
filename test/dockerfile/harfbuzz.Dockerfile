FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/harfbuzz && \
    cd submodule/ && bash ../script/build_harfbuzz.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

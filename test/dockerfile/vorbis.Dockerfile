FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/vorbis && \
    cd submodule/ && bash ../script/build_vorbis.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

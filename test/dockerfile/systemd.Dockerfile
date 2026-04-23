FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/systemd && \
    cd submodule/ && bash ../script/build_systemd.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

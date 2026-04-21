FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/proj4 && \
    cd submodule/ && bash ../script/build_proj4.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

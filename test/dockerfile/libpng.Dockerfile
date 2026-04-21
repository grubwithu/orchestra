FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/libpng && \
    cd submodule/ && bash ../script/build_libpng.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

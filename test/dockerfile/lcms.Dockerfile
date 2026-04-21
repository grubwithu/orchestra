FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/lcms && \
    cd submodule/ && bash ../script/build_lcms.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

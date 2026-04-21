FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/re2 && \
    cd submodule/ && bash ../script/build_re2.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

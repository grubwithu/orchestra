FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/jsoncpp && \
    cd submodule/ && bash ../script/build_jsoncpp.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/sqlite3 && \
    cd submodule/ && bash ../script/build_sqlite3.sh

RUN /usr/local/bin/prepare_hfc_artifacts.sh

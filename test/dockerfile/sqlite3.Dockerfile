FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/sqlite3 && \
    cd submodule/ && bash ../script/build_sqlite3.sh

RUN cd /opt && cd targets/ && bash hfc.sh && cd .. && \
    cd fuzzers/ && bash hfc.sh && mv seeds/* /root/hfc/test/

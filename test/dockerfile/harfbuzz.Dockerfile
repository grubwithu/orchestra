FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/harfbuzz && \
    cd submodule/ && bash ../script/build_harfbuzz.sh

RUN cd /opt && cd targets/ && bash hfc.sh && cd .. && \
    cd fuzzers/ && bash hfc.sh && mv seeds/* /root/hfc/test/

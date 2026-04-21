FROM hfc-test:latest

WORKDIR /root/hfc/test

RUN git submodule update --init --recursive submodule/lcms && \
    cd submodule/ && bash ../script/build_lcms.sh

RUN cd /opt && cd targets/ && bash hfc.sh && cd .. && \
    cd fuzzers/ && bash hfc.sh && mv seeds/* /root/hfc/test/

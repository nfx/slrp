FROM scratch

ENV PWD="/app"
WORKDIR $PWD
COPY slrp $PWD

RUN printf '\
app:\n\
    state: $PWD/.slrp/data\n\
    sync: 1m\n\
log:\n\
    level: info\n\
    format: pretty\n\
server:\n\
    addr: "0.0.0.0:8089"\n\
    read_timeout: 15s\n\
mitm:\n\
    addr: "0.0.0.0:8090"\n\
    read_timeout: 15s\n\
    idle_timeout: 15s\n\
    write_timeout: 15s\n\
checker:\n\
    timeout: 5s\n\
    strategy: simple\n\
history:\n\
    limit: 1000\n\
' > ./slrp.yml
RUN mkdir ./.slrp

EXPOSE 8089 8090

CMD ["./slrp"]

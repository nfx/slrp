FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

RUN echo '\
Acquire::http::No-Cache true;\
APT::Get::Assume-Yes "true";\
APT::Install-Recommends "false";\
APT::Install-Suggests "false";\
' > /etc/apt/apt.conf.d/99custom

RUN apt-get update && apt-get upgrade
RUN apt-get install curl
RUN apt-get install ca-certificates

ENV PWD="/usr/app"
WORKDIR $PWD
ENV version="0.1.5"

RUN curl -sSL "https://github.com/nfx/slrp/releases/download/v$version/slrp_${version}_linux_amd64.tar.gz" | tar xfz -

RUN mkdir ./.slrp
RUN echo '\
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
  write_timeout: 15s\
' > ./slrp.yml

EXPOSE 8089 8090

CMD ["./slrp"]

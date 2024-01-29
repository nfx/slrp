FROM golang:1.20.13-bookworm AS GO_BUILD

WORKDIR /app

COPY . .

RUN make build-go-for-docker

FROM node:20.11-bookworm AS NODE_BUILD

WORKDIR /ui

COPY ./ui .

RUN npm install

# Final image
FROM alpine:latest

# SLRP configuration environment variables
ENV SLRP_APP_STATE="$PWD/.slrp/data"
ENV SLRP_APP_SYNC="1m"
ENV SLRP_LOG_LEVEL="info"
ENV SLRP_LOG_FORMAT="pretty"
ENV SLRP_SERVER_ADDR="0.0.0.0:8089"
ENV SLRP_SERVER_READ_TIMEOUT="15s"
ENV SLRP_MITM_ADDR="0.0.0.0:8090"
ENV SLRP_MITM_READ_TIMEOUT="15s"
ENV SLRP_MITM_IDLE_TIMEOUT="15s"
ENV SLRP_MITM_WRITE_TIMEOUT="15s"
ENV SLRP_CHECKER_TIMEOUT="5s"
ENV SLRP_CHECKER_STRATEGY="simple"
ENV SLRP_HISTORY_LIMIT="1000"

ENV PWD="/opt"
WORKDIR $PWD

COPY --from=GO_BUILD /app/main $PWD/slrp
COPY --from=NODE_BUILD /ui $PWD/ui/

RUN mkdir ./.slrp

EXPOSE 8089 8090

# The new script requires iproute2 via apk
RUN apk add --no-cache iproute2
# Copy the script
COPY ./docker-entrypoint.sh $PWD/docker-entrypoint.sh
# Make the script executable
RUN chmod +x $PWD/docker-entrypoint.sh

# Run the script
ENTRYPOINT ["/opt/docker-entrypoint.sh"]

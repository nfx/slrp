# Install node and deps to build the frontend
FROM node:20.11-bookworm AS NODE_INSTALL
WORKDIR /app
COPY . .
RUN npm --prefix ui install && \
    npm --prefix ui run build

# Install go and deps to build the backend
FROM golang:1.20.13-bookworm AS BUILD
WORKDIR /app
COPY --from=NODE_INSTALL /app .
RUN make build-go-for-docker

# Final image
FROM alpine:latest
# SLRP configuration environment variables
ENV SLRP_APP_STATE="/opt/.slrp/data" \
    SLRP_APP_SYNC="1m" \
    SLRP_LOG_LEVEL="info" \
    SLRP_LOG_FORMAT="pretty" \
    SLRP_SERVER_ADDR="0.0.0.0:8089" \
    SLRP_SERVER_READ_TIMEOUT="15s" \
    SLRP_MITM_ADDR="0.0.0.0:8090" \
    SLRP_MITM_READ_TIMEOUT="15s" \
    SLRP_MITM_IDLE_TIMEOUT="15s" \
    SLRP_MITM_WRITE_TIMEOUT="15s" \
    SLRP_CHECKER_TIMEOUT="5s" \
    SLRP_CHECKER_STRATEGY="simple" \
    SLRP_HISTORY_LIMIT="1000"
WORKDIR /opt
COPY --from=BUILD /app/main /opt/slrp
RUN mkdir -p ./.slrp/data
EXPOSE 8089 8090
CMD ["/opt/slrp"]
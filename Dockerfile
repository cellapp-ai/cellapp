FROM node:22-bookworm AS web
WORKDIR /src
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
RUN npm ci
COPY apps/web/ apps/web/
RUN npm run build --workspace @cellapp/web

FROM golang:1.24-alpine AS server
WORKDIR /src
ARG GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum ./
RUN go mod download
COPY apps/server/ apps/server/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cellapp-server ./apps/server/cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cellapp-copy-releases ./apps/server/cmd/copy-releases && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cloudrun-entrypoint ./apps/server/cmd/cloudrun
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cellapp-healthcheck ./apps/server/cmd/healthcheck

FROM caddy:2.10.2-alpine AS caddy
FROM ghcr.io/postgrest/postgrest:v16.3@sha256:ec0e25a4e24b0a3bc5e4f011369bfc736bd1b19f513bd01079b86329a7636962 AS postgrest

FROM alpine:3.22
COPY --from=caddy /usr/bin/caddy /usr/bin/caddy
COPY --from=postgrest /bin/postgrest /usr/local/bin/postgrest
COPY --from=caddy /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -S cellapp && adduser -S -G cellapp cellapp
COPY --from=server /out/cellapp-server /usr/local/bin/cellapp-server
COPY --from=server /out/cellapp-copy-releases /usr/local/bin/cellapp-copy-releases
COPY --from=web /src/apps/web/dist/ /opt/cellapp/web/
COPY infra/Caddyfile.cloudrun /etc/caddy/Caddyfile
COPY --from=server /out/cloudrun-entrypoint /usr/local/bin/cloudrun-entrypoint
COPY --from=server /out/cellapp-healthcheck /usr/local/bin/cellapp-healthcheck
COPY infra/selfhost-entrypoint.sh /usr/local/bin/selfhost-entrypoint
RUN chmod 0555 /usr/local/bin/cloudrun-entrypoint /usr/local/bin/selfhost-entrypoint /usr/local/bin/cellapp-server /usr/local/bin/cellapp-healthcheck /usr/local/bin/cellapp-copy-releases /usr/local/bin/postgrest && chown -R cellapp:cellapp /opt/cellapp
USER cellapp
ENV NODE_ENV=production WEB_ROOT=/opt/cellapp/web LISTEN_ADDRESS=127.0.0.1:3000
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/cloudrun-entrypoint"]

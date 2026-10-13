FROM node:22-bookworm AS web
WORKDIR /src
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
RUN npm ci
COPY apps/web/ apps/web/
RUN npm run build --workspace @cellapp/web

FROM golang:1.24-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps/server/ apps/server/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cellapp-server ./apps/server/cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cloudrun-entrypoint ./apps/server/cmd/cloudrun

FROM caddy:2.10.2-alpine AS caddy

FROM alpine:3.22
COPY --from=caddy /usr/bin/caddy /usr/bin/caddy
COPY --from=caddy /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -S cellapp && adduser -S -G cellapp cellapp
COPY --from=server /out/cellapp-server /usr/local/bin/cellapp-server
COPY --from=web /src/apps/web/dist/ /opt/cellapp/web/
COPY infra/Caddyfile.cloudrun /etc/caddy/Caddyfile
COPY --from=server /out/cloudrun-entrypoint /usr/local/bin/cloudrun-entrypoint
RUN chmod 0555 /usr/local/bin/cloudrun-entrypoint /usr/local/bin/cellapp-server && chown -R cellapp:cellapp /opt/cellapp
USER cellapp
ENV NODE_ENV=production WEB_ROOT=/opt/cellapp/web LISTEN_ADDRESS=127.0.0.1:3000
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/cloudrun-entrypoint"]

# Builds both shop_notifier binaries into one small, non-root image — see
# shop_ingestor's Dockerfile for why one image, selected by command at
# deploy time.

FROM golang:1.25.14-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/notifier ./cmd/notifier && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/notifier /out/migrate ./
USER nonroot:nonroot
ENTRYPOINT ["/app/notifier"]

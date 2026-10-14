FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/fudia-image-processor ./cmd/processor

FROM builder AS test-builder
RUN CGO_ENABLED=0 go test -c -o /out/imaging.test ./internal/infrastructure/imaging

FROM alpine:3.22 AS runtime-base
RUN apk add --no-cache ca-certificates tzdata vips-tools && addgroup -S app && adduser -S -G app app
WORKDIR /app
ENV GOMEMLIMIT=512MiB
USER app

FROM runtime-base AS verification
COPY --from=test-builder /out/imaging.test /app/imaging.test
ENTRYPOINT ["/app/imaging.test"]

FROM runtime-base AS runtime
COPY --from=builder /out/fudia-image-processor /app/fudia-image-processor
EXPOSE 8080
ENTRYPOINT ["/app/fudia-image-processor"]


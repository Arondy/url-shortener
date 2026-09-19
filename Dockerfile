FROM golang:1.26.8-alpine3.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /url-shortener ./cmd/url-shortener

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder --chmod=755 /url-shortener /url-shortener

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/url-shortener"]


FROM golang:1.27.2-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/circuito-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/circuito-api /circuito-api

USER nonroot:nonroot

EXPOSE 8181

ENTRYPOINT ["/circuito-api"]

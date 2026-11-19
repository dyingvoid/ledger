FROM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/ledger ./cmd/ledger

FROM gcr.io/distroless/cc-debian12:nonroot

COPY --from=build /out/ledger /ledger

ENTRYPOINT ["/ledger"]

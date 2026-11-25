FROM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY shared/go.mod shared/go.sum ./shared/
RUN go -C shared mod download

COPY services/accounting/go.mod services/accounting/go.sum ./services/accounting/
RUN go -C services/accounting mod download

COPY shared ./shared
COPY services/accounting ./services/accounting

RUN CGO_ENABLED=1 go -C services/accounting build -trimpath -ldflags="-s -w" -o /out/ledger ./cmd/ledger

FROM gcr.io/distroless/cc-debian12:nonroot

COPY --from=build /out/ledger /ledger

ENTRYPOINT ["/ledger"]

FROM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY go.work ./
COPY services/accounting/go.mod services/accounting/go.sum ./services/accounting/
COPY services/bff/go.mod services/bff/go.sum ./services/bff/

RUN go work sync

COPY services/accounting ./services/accounting
COPY services/bff ./services/bff

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/bff ./services/bff/cmd/bff

FROM gcr.io/distroless/cc-debian12:nonroot

COPY --from=build /out/bff /bff

ENTRYPOINT ["/bff"]

FROM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY shared/go.mod shared/go.sum ./shared/
RUN go -C shared mod download

COPY services/bff/go.mod services/bff/go.sum ./services/bff/
RUN go -C services/bff mod download

COPY shared ./shared
COPY services/bff ./services/bff

RUN CGO_ENABLED=1 go -C services/bff build -trimpath -ldflags="-s -w" -o /out/bff ./cmd/bff

FROM gcr.io/distroless/cc-debian12:nonroot

COPY --from=build /out/bff /bff

ENTRYPOINT ["/bff"]

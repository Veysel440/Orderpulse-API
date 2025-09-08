FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/orderpulse-api ./cmd/orderpulse-api

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/orderpulse-api /orderpulse-api
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/orderpulse-api"]
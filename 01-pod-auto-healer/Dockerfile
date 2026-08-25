FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/auto-healer ./cmd/auto-healer

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/auto-healer /auto-healer
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/auto-healer"]

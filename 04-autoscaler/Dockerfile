FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/autoscaler ./cmd/autoscaler

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/autoscaler /autoscaler
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/autoscaler"]

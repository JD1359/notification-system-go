FROM golang:1.22-alpine AS build
WORKDIR /src

# Copy go.mod (and go.sum if it exists). Then resolve & download deps -
# this also generates go.sum inside the container if it's missing locally.
COPY go.mod ./
COPY go.su[m] ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /out/api    ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/api    /api
COPY --from=build /out/worker /worker
EXPOSE 8080
ENTRYPOINT ["/api"]

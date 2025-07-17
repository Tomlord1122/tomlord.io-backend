FROM golang:1.24-alpine AS build

WORKDIR /app

# COPY go.mod go.sum ./ -> Copy go.mod and go.sum to the working directory
COPY go.mod go.sum ./
RUN go mod download
# COPY src to dst -> Copy the source code to the working directory
COPY . .

# go build -o main cmd/api/main.go -> Build the main application
RUN go build -o main cmd/api/main.go

# FROM alpine:3.20.1 AS prod -> Create a new stage from the alpine image
FROM alpine:3.20.1 AS prod
WORKDIR /app
# COPY --from=build /app/main /app/main -> Copy the main application from the build stage to the working directory
COPY --from=build /app/main /app/main
# EXPOSE ${PORT} -> Expose the port
EXPOSE ${PORT}
# CMD ["./main"] -> Run the main application
CMD ["./main"]






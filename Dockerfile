# Use the official Golang image to create a build artifact.
FROM golang:1.23-alpine

WORKDIR /app
COPY . .
RUN go build -o workorder

EXPOSE 8080
CMD ["./workorder"]

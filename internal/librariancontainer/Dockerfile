FROM golang:1.24

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY . /app
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o librariancontainer

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/reference/dockerfile/#expose
EXPOSE 8080

ENTRYPOINT ["/app/librariancontainer"]

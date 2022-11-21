FROM golang

WORKDIR /server

ENV HTTP_PORT 8088

# Copy go files
COPY go.mod ./
COPY go.sum ./
COPY *.go ./

# Download dependencies
RUN go mod download

# Build and start
RUN go build -o /server-build
EXPOSE 8088
CMD ["/server-build"]
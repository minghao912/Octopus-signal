FROM golang

WORKDIR /server

# Copy go files
COPY go.mod ./
COPY go.sum ./
COPY *.go ./

# Download dependencies
RUN go mod download

# Build and start
RUN go build -o /server-build
EXPOSE 8080
CMD ["/server-build"]
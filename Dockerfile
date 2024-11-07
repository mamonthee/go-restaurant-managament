# Use the official Golang image to build the app
FROM golang:1.20 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Copy the .env file into the container (optional)
COPY .env ./

# Build the Go app
RUN go build -o main .

# Use a smaller base image to run the app
FROM gcr.io/distroless/base

# Copy the binary and .env file from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
COPY --from=builder /app/frontend/dist ./frontend/dist  

# Set the port to expose
EXPOSE 9000

# Command to run the executable
CMD ["./main"]

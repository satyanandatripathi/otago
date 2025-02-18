# Use the official Golang image as the base image
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o main .
EXPOSE 5000
# Command to run the application
CMD ["./main"]
# Vroom Vroom

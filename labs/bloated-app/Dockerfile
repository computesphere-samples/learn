# This Dockerfile works, but it is deliberately wasteful. Lesson 3.4.5 fixes it.
# Count the problems before you read the lesson:

# The full Go toolchain image: compiler, git, build tools, a whole Debian.
FROM golang:1.27

# Build and run in the same folder, as root (no USER line anywhere).
WORKDIR /app

# Copies everything in this folder, and there's no .dockerignore to filter it.
COPY . .

RUN go build -o server .

EXPOSE 3000

# The compiled binary is a few MB, but it ships inside the whole build image.
CMD ["./server"]

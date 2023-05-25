FROM ubuntu

# Set the Current Working Directory inside the container

RUN apt update

RUN mkdir /app
COPY voedger /app

# This container exposes port 8888 to the outside world
EXPOSE 8888

# Run the executable
CMD ["/app/voedger"]






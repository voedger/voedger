FROM ubuntu:20.04

# run something like: docker run -e VOEDGER_HTTP_PORT=8888 voedger/voedger:0.0.1-alpha

RUN apt update

RUN mkdir /app
COPY voedger /app

# Set the default HTTP port may be override with -e option in docker
ENV VOEDGER_HTTP_PORT 80

# This container exposes port 8888 to the outside world
EXPOSE $VOEDGER_HTTP_PORT

# Set the Current Working Directory inside the container
WORKDIR /app

# Set default command
CMD exec /app/voedger server --ihttp.Port=$VOEDGER_HTTP_PORT

FROM golang:alpine
RUN mkdir /app 
ADD . /app/
WORKDIR /app 
ARG channeldb
ENV OZIACH_CHANNEL_DB=${channeldb}
ARG auth
ENV OZIACH_AUTH=${auth}
ARG access
ENV AWS_ACCESS_KEY_ID=${access}
ARG secret
ENV AWS_SECRET_ACCESS_KEY=${secret}
CMD go run main.go
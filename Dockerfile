FROM golang:alpine
RUN mkdir /app 
ADD . /app/
WORKDIR /app 
ARG auth
ENV OZIACH_AUTH=${auth}
CMD go run main.go
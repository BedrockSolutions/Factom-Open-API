FROM golang:1.12

RUN curl https://glide.sh/get | sh

ENV PKG_NAME=github.com/DeFacto-Team/Factom-Open-API
ENV PKG_PATH=$GOPATH/src/$PKG_NAME
WORKDIR $PKG_PATH

COPY glide.yaml glide.lock $PKG_PATH/
RUN glide install -v

COPY . $PKG_PATH
RUN go build main.go

RUN mkdir -p /.factom-open-api
COPY config.yaml /.factom-open-api/

CMD ["./main"]
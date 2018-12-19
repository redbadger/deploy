#~~~~~~~~~~~~~~~~~~~~~~~
FROM golang:1.10-alpine as base
RUN apk --no-cache add \
  ca-certificates \
  curl \
  git \
  ;

#~~~~~~~~~~~~~~~~~~~~~~~
FROM base as builder

WORKDIR /go/src/github.com/redbadger/deploy
COPY . .
RUN go get -d ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deploy .

ENV KUBE_LATEST_VERSION="v1.9.5"
RUN curl -L https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
  && chmod +x /usr/local/bin/kubectl

#~~~~~~~~~~~~~~~~~~~~~~~
FROM base as release

WORKDIR /root/
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin
COPY --from=builder /go/src/github.com/redbadger/deploy/deploy .
ENTRYPOINT ["./deploy"]
CMD ["help"]

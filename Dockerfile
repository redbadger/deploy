#~~~~~~~~~~~~~~~~~~~~~~~
FROM golang:1.11.2-alpine3.8 as builder
RUN apk --no-cache add \
  git \
  curl

WORKDIR /src
COPY . .
RUN go get -d ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deploy .

ENV KUBE_LATEST_VERSION="v1.12.3"
RUN curl -L https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
  && chmod +x /usr/local/bin/kubectl

#~~~~~~~~~~~~~~~~~~~~~~~
FROM alpine:3.8
RUN apk --no-cache add \
  ca-certificates \
  git \
  ;

WORKDIR /root/
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin
COPY --from=builder /src/deploy .
ENTRYPOINT ["./deploy"]
CMD ["help"]

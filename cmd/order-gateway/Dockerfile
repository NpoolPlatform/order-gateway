FROM uhub.service.ucloud.cn/entropypool/alpine:3.16

WORKDIR /app

COPY order-gateway /app

ENTRYPOINT ["/app/order-gateway"]

EXPOSE 50430 50431 50432 50433

CMD ["run"]

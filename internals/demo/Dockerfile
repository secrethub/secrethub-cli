FROM alpine

RUN apk add --repository https://alpine.secrethub.io/alpine/edge/main --allow-untrusted secrethub-cli

EXPOSE 80
ENTRYPOINT ["secrethub", "run", "--"]
CMD ["secrethub", "demo", "serve", "--host", "0.0.0.0", "--port", "80"]

ADD secrethub.env .

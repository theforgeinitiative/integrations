FROM golang:1.19-alpine as builder

WORKDIR /app

COPY . ./
RUN go mod download

RUN go build -o forgebot cmd/bot/bot.go

FROM alpine
COPY --from=builder /app/forgebot /bin/forgebot

CMD [ "/bin/forgebot" ]
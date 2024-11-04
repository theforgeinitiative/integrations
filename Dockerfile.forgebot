FROM golang:1.19-alpine as builder

WORKDIR /app

COPY . ./
RUN go mod download

RUN go build -o forgebot cmd/bot/bot.go

FROM alpine
RUN apk add --no-cache tzdata
ENV TZ=America/New_York
COPY --from=builder /app/forgebot /bin/forgebot

CMD [ "/bin/forgebot" ]
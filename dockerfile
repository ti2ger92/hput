FROM golang:1.18.0-buster

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

CMD ["go", "run", "/usr/src/app/main/main.go", "-nonlocal", "True"]

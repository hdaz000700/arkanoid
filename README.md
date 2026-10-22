sudo go mod tidy

sudo GOOS=linux GOARCH=amd64 go build -o arkanoid main.go

./arkanoid

build:
    go build -o att .

test:
    go vet ./...
    go test ./...

# Install into ~/.local/bin (on PATH, stable path for the service)
install:
    GOBIN="$HOME/.local/bin" go install .

# Cross-compile release binaries into dist/
dist:
    GOOS=darwin GOARCH=arm64 go build -o dist/att-darwin-arm64 .
    GOOS=darwin GOARCH=amd64 go build -o dist/att-darwin-amd64 .
    GOOS=linux  GOARCH=amd64 go build -o dist/att-linux-amd64 .
    GOOS=linux  GOARCH=arm64 go build -o dist/att-linux-arm64 .

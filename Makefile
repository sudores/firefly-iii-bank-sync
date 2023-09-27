build:
	go build -o app
run: build
	./scripts/startup.sh

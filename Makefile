BINARY = arbot # binary exec name

test: 
	go test -v -cover -covermode=atomic ./...

build:
	go build -o ${BINARY} *.go

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

run-server: ## Starts docker containers for local development.
	docker-compose up --build
	@echo Running arbot development server

stop-server: 
	docker-compose down

.PHONY: clean test build run-docker
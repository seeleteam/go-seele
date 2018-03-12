all: discovery node
discovery:
	go build -o ./build/discovery ./cmd/discovery
	@echo "Done discovery building"

node:
	go build -o ./build/node ./cmd/node 
	@echo "Done node building"

.PHONY: discovery node

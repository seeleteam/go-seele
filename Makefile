all: discovery node
discovery:
	go build ./cmd/discovery
	@echo "Done discovery building"

node:
	go build -o seele-node ./cmd/node 
	@echo "Done node building"

.PHONY: discovery node

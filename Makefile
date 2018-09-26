# Makefile to build the command lines and tests in Seele project.
# This Makefile doesn't consider Windows Environment. If you use it in Windows, please be careful.

all: discovery node client light tool vm
discovery:
	go build -o ./build/discovery ./cmd/discovery
	@echo "Done discovery building"

node:
	go build -o ./build/node ./cmd/node 
	@echo "Done node building"

client:
	go build -o ./build/client ./cmd/client
	@echo "Done full node client building"

light:
	go build -o ./build/light ./cmd/client/light
	@echo "Done light node client building"

tool:
	go build -o ./build/tool ./cmd/tool
	@echo "Done tool building"

vm:
	go build -o ./build/vm ./cmd/vm
	@echo "Done vm building"

.PHONY: discovery node client light tool vm

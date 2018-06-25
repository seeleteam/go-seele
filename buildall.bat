@echo off 
goto comment
    Build the command lines and tests in Windows.
    Must install gcc tool before building.
:comment

echo on

go build -o ./build/discovery.exe ./cmd/discovery
@echo "Done discovery building"

go build -o ./build/node.exe ./cmd/node 
@echo "Done node building"

go build -o ./build/client.exe ./cmd/client
@echo "Done client building"

go build -o ./build/tool.exe ./cmd/tool
@echo "Done tool building"

go build -o ./build/vm.exe ./cmd/vm
@echo "Done vm building"

pause
@echo off
goto comment
    Build the command lines and tests in Windows.
    Must install gcc tool before building.
:comment

set para=%*
if not defined para (
    set para=all
)

for %%i in (%para%) do (
    call :%%i
)
pause
goto:eof

:all
call :discovery
call :node
call :client
call :light
call :tool
call :vm
goto:eof

:discovery
echo on
go build -o ./build/discovery.exe ./cmd/discovery
@echo "Done discovery building"
@echo off
goto:eof

:node
echo on
go build -o ./build/node.exe ./cmd/node
@echo "Done node building"
@echo off
goto:eof

:client
echo on
go build -o ./build/client.exe ./cmd/client
@echo "Done full node client building"
@echo off
goto:eof

:light
echo on
go build -o ./build/light.exe ./cmd/client/light
@echo "Done light node client building"
@echo off
goto:eof

:tool
echo on
go build -o ./build/tool.exe ./cmd/tool
@echo "Done tool building"
@echo off
goto:eof

:vm
echo on
go build -o ./build/vm.exe ./cmd/vm
@echo "Done vm building"
@echo off
goto:eof

:clean
del build\* /q /f /s
@echo "Done clean the build dir"
@echo off
goto:eof
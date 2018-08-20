@echo off 
goto comment
    Build the command lines and tests in Windows.
    Must install gcc tool before building.
:comment

set para=%*
if not defined para (
    set act=all
)else (
    set act=%1
)

echo "choose cmd [%act%]"

if "%act%"=="all" (
    call :all 
) else ( 
    if "%act%" == "discovery" (
        call :discovery
    ) else (
        if "%act%" == "node" (
                call :node
        ) else (
            if "%act%" == "client" (
                call :client
            ) else (
                if "%act%" == "tool" (
                    call :tool
                ) else (
                     if "%act%" == "vm" (
                        call :vm
                    ) else (
                        if "%act%" == "clean" (
                            call :clean
                        ) else (
                            echo "error cmd %act%"
                        )
                    )
                )
            )
        )
    )
)
exit

:all 
call :discovery
call :node 
call :client 
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
@echo "Done client building"
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
echo "call clean"
echo "clean the build dir"
del build\* /q /f /s
@echo off
goto:eof

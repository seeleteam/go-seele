start node.exe start -c ..\tool\config\node1.json --accounts ..\tool\accounts.json
ping -n 3 127.0.0.1>nul
start node.exe start -c ..\tool\config\node2.json --accounts ..\tool\accounts.json
start node.exe start -c ..\tool\config\node3.json --accounts ..\tool\accounts.json
@echo off
mkdir bin
set GOPATH=%cd%
go build -o %GOPATH%\bin\slic3rServer.exe %GOPATH%\src\main.go
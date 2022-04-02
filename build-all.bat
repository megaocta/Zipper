@echo off
SET GOOS=windows
go build -o zipper.exe

SET GOOS=linux
go build -o zipper

SET GOARCH=arm
SET GOOS=linux
go build -o zipper_armhf

start upx -9 zipper.exe
start upx -9 zipper
start upx -9 zipper_armhf

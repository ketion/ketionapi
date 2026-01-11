@echo off
chcp 65001 >nul
echo ========================================
echo 预先拉取 Docker 基础镜像
echo ========================================
echo.

echo 正在拉取 oven/bun:latest...
docker pull oven/bun:latest

echo.
echo 正在拉取 golang:alpine...
docker pull golang:alpine

echo.
echo 正在拉取 debian:bookworm-slim...
docker pull debian:bookworm-slim

echo.
echo ========================================
echo 镜像拉取完成！
echo ========================================
echo.
echo 现在可以运行 build-and-export.bat 进行构建
echo.

pause

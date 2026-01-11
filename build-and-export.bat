@echo off
chcp 65001 >nul
echo ========================================
echo 构建并导出 New-API Docker 镜像
echo （支持魔塔平台文生图）
echo ========================================
echo.

REM 读取版本号，如果为空则使用默认值
set VERSION=
set /p VERSION=<VERSION
if "%VERSION%"=="" (
    set VERSION=1.0.0
    echo VERSION 文件为空，使用默认版本: %VERSION%
) else (
    echo 当前版本: %VERSION%
)
echo.

REM 设置镜像名称
set IMAGE_NAME=new-api-modelscope
set EXPORT_FILE=new-api-modelscope-%VERSION%.tar

echo 开始构建 Docker 镜像...
echo.
echo 提示: 如果看到 git 相关警告可以忽略，不影响构建
echo.
docker build -t %IMAGE_NAME%:latest -t %IMAGE_NAME%:%VERSION% .

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ 镜像构建失败！
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo ✅ 镜像构建成功！
echo ========================================
echo.

REM 询问是否导出镜像
set /p EXPORT_CHOICE=是否导出镜像文件用于服务器部署？(Y/N): 
if /i "%EXPORT_CHOICE%"=="Y" (
    echo.
    echo 正在导出镜像到 %EXPORT_FILE%...
    docker save -o %EXPORT_FILE% %IMAGE_NAME%:latest
    
    if %ERRORLEVEL% EQU 0 (
        echo.
        echo ✅ 镜像已导出到: %EXPORT_FILE%
        echo.
        echo 文件大小:
        dir %EXPORT_FILE% | find "%EXPORT_FILE%"
    ) else (
        echo.
        echo ❌ 镜像导出失败！
    )
)

echo.
echo ========================================
echo 部署步骤：
echo ========================================
echo.
echo 1. 上传文件到服务器:
echo    - %EXPORT_FILE%
echo    - docker-compose.yml
echo.
echo 2. 在宝塔面板中操作:
echo    a. 安装 Docker 和 Docker Compose
echo    b. 进入终端，导入镜像:
echo       docker load -i %EXPORT_FILE%
echo.
echo 3. 启动服务:
echo    docker-compose up -d
echo.
echo 4. 访问地址:
echo    http://你的服务器IP:3000
echo.
echo 默认账号: root
echo 默认密码: 123456
echo.

pause

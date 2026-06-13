@echo off
chcp 65001
echo ========================================
echo    古代石窟风化监测系统 - 启动脚本
echo ========================================
echo.

echo [1/4] 启动 TimescaleDB 数据库...
docker compose up -d timescaledb
if errorlevel 1 (
    echo 警告: Docker 启动失败，请确保 Docker Desktop 已运行
    echo 如果已安装 PostgreSQL，可跳过此步骤
)

echo.
echo 等待数据库启动...
timeout /t 15

echo.
echo [2/4] 下载 Go 依赖...
cd backend
go mod tidy
cd ..

echo.
echo [3/4] 启动 Go 后端服务...
start "Grotto Backend" cmd /k "cd backend && go run main.go"

echo.
echo [4/4] 启动前端服务...
start "Grotto Frontend" cmd /k "cd frontend && python -m http.server 8000"

echo.
echo ========================================
echo  服务启动完成！
echo ========================================
echo  后端 API: http://localhost:8080
echo  前端界面: http://localhost:8000
echo  数据库:   localhost:5432
echo ========================================
echo.
echo 如需启动传感器模拟器，请运行: start-simulator.bat
echo.
pause

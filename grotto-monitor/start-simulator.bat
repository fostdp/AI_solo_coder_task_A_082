@echo off
chcp 65001
echo ========================================
echo    传感器数据模拟器
echo ========================================
echo.

cd backend\simulator

echo 启动传感器数据模拟器...
echo 注意: 模拟器会先回灌 30 天历史数据，然后每小时上报一次
echo.

go run simulator.go

pause

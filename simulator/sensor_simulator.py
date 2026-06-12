#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
古代石窟风化监测系统 - 传感器数据模拟器
模拟每小时通过4G DTU上报微环境和监测数据到后端API
"""

import argparse
import json
import math
import random
import sys
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional

try:
    import requests
except ImportError:
    print("请先安装 requests 库: pip install requests")
    sys.exit(1)


ROCK_BASE_PARAMS = {
    "砂砾岩": {"base_temp": 12.0, "base_hum": 40.0, "base_hardness": 38.0, "hardness_var": 12.0},
    "砂岩":   {"base_temp": 14.0, "base_hum": 50.0, "base_hardness": 52.0, "hardness_var": 14.0},
    "石灰岩": {"base_temp": 17.0, "base_hum": 60.0, "base_hardness": 62.0, "hardness_var": 10.0},
    "花岗岩": {"base_temp": 13.0, "base_hum": 45.0, "base_hardness": 80.0, "hardness_var": 15.0},
}


class SensorSimulator:
    def __init__(self, api_base: str = "http://localhost:8080/api/v1", dry_run: bool = False):
        self.api_base = api_base.rstrip("/")
        self.dry_run = dry_run
        self.caves: List[Dict] = []
        self.monitoring_points: List[Dict] = []
        self.point_states: Dict[int, Dict] = {}
        self.session = requests.Session()

    def _get(self, path: str) -> Optional[Dict]:
        url = f"{self.api_base}{path}"
        try:
            if self.dry_run:
                return {"code": 0, "data": []}
            resp = self.session.get(url, timeout=15)
            resp.raise_for_status()
            data = resp.json()
            if data.get("code") == 0:
                return data
            print(f"[WARN] API返回错误: {data.get('message')}")
            return data
        except requests.RequestException as e:
            print(f"[ERROR] 请求失败 {url}: {e}")
            return None

    def _post(self, path: str, payload) -> Optional[Dict]:
        url = f"{self.api_base}{path}"
        try:
            if self.dry_run:
                return {"code": 0, "data": {"received": True}}
            resp = self.session.post(url, json=payload, timeout=15)
            resp.raise_for_status()
            return resp.json()
        except requests.RequestException as e:
            print(f"[ERROR] 提交失败 {url}: {e}")
            return None

    def fetch_metadata(self) -> bool:
        print("[INFO] 获取石窟和监测点元数据...")
        caves_resp = self._get("/caves")
        if caves_resp:
            self.caves = caves_resp.get("data") or []
            print(f"  ✓ 获取到 {len(self.caves)} 处石窟")
        else:
            self._generate_mock_caves()

        if self.caves:
            cave_ids = [c["id"] for c in self.caves[:3]]
            for cid in cave_ids:
                pt_resp = self._get(f"/points?caveId={cid}")
                pts = pt_resp.get("data") or [] if pt_resp else []
                self.monitoring_points.extend(pts)
                print(f"  ✓ 石窟#{cid} 获取到 {len(pts)} 个监测点")

            if not self.monitoring_points:
                self._generate_mock_points()
        else:
            self._generate_mock_points()

        print(f"  共 {len(self.monitoring_points)} 个监测点将参与模拟")
        self._init_point_states()
        return True

    def _generate_mock_caves(self):
        print("[WARN] 使用模拟石窟数据")
        rock_types = ["砂砾岩", "砂岩", "石灰岩"]
        provinces = ["甘肃", "山西", "河南", "重庆", "新疆", "河北"]
        for i in range(10):
            self.caves.append({
                "id": i + 1,
                "name": f"模拟石窟#{i+1}",
                "province": random.choice(provinces),
                "rockType": random.choice(rock_types),
            })

    def _generate_mock_points(self):
        print("[WARN] 使用模拟监测点数据")
        pt_id = 1
        for cave in self.caves[:5]:
            for i in range(10):
                self.monitoring_points.append({
                    "id": pt_id,
                    "caveId": cave["id"],
                    "name": f"{cave['name']}监测点#{str(i+1).zfill(3)}",
                    "code": f"MP{pt_id}",
                    "rockType": cave["rockType"],
                    "initialHardness": ROCK_BASE_PARAMS.get(cave["rockType"], {})["base_hardness"] + random.uniform(-5, 5),
                    "initialCrackWidth": 0.0 if random.random() > 0.4 else round(random.uniform(0.05, 0.3), 4),
                    "positionX": round(random.uniform(-50, 50), 2),
                    "positionY": round(random.uniform(-50, 50), 2),
                    "positionZ": round(random.uniform(0, 50), 2),
                })
                pt_id += 1

    def _init_point_states(self):
        for pt in self.monitoring_points:
            rock_params = ROCK_BASE_PARAMS.get(pt.get("rockType", "砂岩"), ROCK_BASE_PARAMS["砂岩"])
            self.point_states[pt["id"]] = {
                "point": pt,
                "params": rock_params,
                "current_hardness": pt.get("initialHardness", 50.0),
                "current_crack": pt.get("initialCrackWidth", 0.0),
                "total_hours": 0,
            }

    def _calc_weathering_factors(self, state: Dict, temp: float, hum: float, hour_offset: int) -> Dict:
        params = state["params"]
        initial_hardness = state["point"].get("initialHardness", 50.0)

        t = hour_offset / 8760.0

        temp_stress = math.pow(abs(temp - 15), 1.5) * 0.0015
        hum_stress = math.pow(hum - 60, 2) * 0.0004
        if hum < 30:
            hum_stress += (30 - hum) * 0.008
        interaction = 0.0
        if temp > 32 and hum > 75:
            interaction = 0.04
        if temp < -3 and hum > 60:
            interaction += 0.06
        freeze_thaw = 0.0
        if -5 < temp < 5:
            freeze_thaw = 0.03 + random.uniform(-0.005, 0.01)

        rock_factor = {"砂砾岩": 1.8, "砂岩": 1.4, "石灰岩": 1.0, "花岗岩": 0.6}
        rf = rock_factor.get(state["point"].get("rockType", "砂岩"), 1.2)

        daily_rate = (temp_stress + hum_stress + interaction + freeze_thraw) if 'freeze_thaw' in dir() else (temp_stress + hum_stress + interaction)
        daily_rate = (temp_stress + hum_stress + interaction + (freeze_thaw if freeze_thaw else 0)) * rf

        hardness_loss = min(initial_hardness * 0.001, daily_rate * initial_hardness * 0.12) / 24
        crack_growth = max(0.0, daily_rate * 0.004 + random.uniform(-0.0005, 0.0008))

        return {"hardness_loss": hardness_loss, "crack_growth": crack_growth}

    def generate_single_reading(self, point_id: int, sim_time: datetime) -> Dict:
        state = self.point_states.get(point_id)
        if not state:
            return {}

        pt = state["point"]
        params = state["params"]

        day_of_year = sim_time.timetuple().tm_yday
        hour = sim_time.hour

        seasonal_temp = 12 * math.sin(2 * math.pi * day_of_year / 365 - math.pi / 2)
        daily_temp = 5 * math.sin(2 * math.pi * hour / 24 - math.pi / 2)
        temperature = params["base_temp"] + seasonal_temp + daily_temp + random.uniform(-2, 2)

        seasonal_hum = -8 * math.sin(2 * math.pi * day_of_year / 365 - math.pi / 2) * 0.35
        daily_hum = -5 * math.sin(2 * math.pi * hour / 24 - math.pi / 2) * 0.25
        humidity = params["base_hum"] + seasonal_hum + daily_hum + random.uniform(-6, 6)
        humidity = max(5.0, min(98.0, humidity))

        hour_offset = state["total_hours"]
        factors = self._calc_weathering_factors(state, temperature, humidity, hour_offset)

        state["current_hardness"] = max(
            10.0,
            state["current_hardness"] - factors["hardness_loss"] + random.uniform(-0.15, 0.1),
        )
        state["current_crack"] = max(
            0.0,
            state["current_crack"] + factors["crack_growth"] + random.uniform(-0.002, 0.003),
        )

        state["total_hours"] += 1

        return {
            "time": sim_time.strftime("%Y-%m-%dT%H:%M:%S+08:00"),
            "pointId": point_id,
            "temperature": round(temperature, 2),
            "humidity": round(humidity, 2),
            "surfaceHardness": round(state["current_hardness"], 2),
            "crackWidth": round(state["current_crack"], 4),
            "windSpeed": round(random.uniform(0, 8), 2),
            "rainfall": round(max(0, random.gauss(0, 3)) if random.random() < 0.04 else 0.0, 2),
            "solarRadiation": round(max(0, 600 * math.sin(2 * math.pi * hour / 24 - math.pi / 2) + random.uniform(-80, 80)), 2),
            "co2Concentration": round(400 + random.uniform(-50, 150), 2),
            "vibration": round(max(0, random.expovariate(40)), 4),
        }

    def send_batch(self, readings: List[Dict]) -> bool:
        if not readings:
            return True
        resp = self._post("/sensors/batch", readings)
        if resp and resp.get("code") == 0:
            data = resp.get("data") or {}
            print(f"  ✓ 提交成功 {data.get('success', 0)}/{data.get('total', len(readings))} 条")
            return True
        else:
            print(f"  ✗ 提交失败: {resp}")
            return False

    def run_historical(self, days: int = 365, batch_size: int = 200):
        print(f"\n{'='*60}")
        print(f"[历史模式] 生成最近 {days} 天的历史数据...")
        print(f"{'='*60}")

        start_time = datetime.now() - timedelta(days=days)
        total_hours = days * 24
        total_points = len(self.monitoring_points)
        total_records = total_hours * total_points
        print(f"  监测点: {total_points}, 总小时: {total_hours}, 预计记录数: ~{total_records:,}")

        current_time = start_time
        batch = []
        sent = 0
        start_clock = time.time()
        report_interval = max(1, total_hours // 50)

        hour_idx = 0
        while current_time <= datetime.now():
            for pt in self.monitoring_points:
                reading = self.generate_single_reading(pt["id"], current_time)
                if reading:
                    batch.append(reading)

                if len(batch) >= batch_size:
                    self.send_batch(batch)
                    sent += len(batch)
                    batch = []

            if hour_idx % report_interval == 0:
                progress = hour_idx / max(1, total_hours) * 100
                elapsed = time.time() - start_clock
                rate = sent / max(1, elapsed)
                eta = (total_records - sent) / max(1, rate) / 60
                print(f"[进度] {progress:5.1f}% | 已发送 {sent:,} | {rate:.0f}条/秒 | 预计剩余 {eta:.0f}分钟 | 当前: {current_time.strftime('%Y-%m-%d %H:00')}")

            current_time += timedelta(hours=1)
            hour_idx += 1

        if batch:
            self.send_batch(batch)
            sent += len(batch)

        elapsed = time.time() - start_clock
        print(f"\n[完成] 历史数据生成完毕! 共发送 {sent:,} 条, 耗时 {elapsed:.1f}秒")

    def run_realtime(self, interval_seconds: int = 3600, speed: float = 1.0):
        print(f"\n{'='*60}")
        print(f"[实时模式] 模拟传感器实时上报 (速度 ×{speed:.1f})")
        print(f"  真实间隔: {interval_seconds}秒, 模拟间隔: {interval_seconds/speed:.1f}秒")
        print(f"  按 Ctrl+C 停止")
        print(f"{'='*60}")

        sim_time = datetime.now().replace(minute=0, second=0, microsecond=0)
        batch = []
        try:
            while True:
                for pt in self.monitoring_points:
                    reading = self.generate_single_reading(pt["id"], sim_time)
                    if reading:
                        batch.append(reading)

                if batch:
                    print(f"[{sim_time.strftime('%H:%M')}] 生成 {len(batch)} 条数据", end=" → ")
                    self.send_batch(batch)
                    batch = []

                sim_time += timedelta(hours=1)
                time.sleep(interval_seconds / speed)

                if sim_time.minute == 0 and sim_time.hour == 0:
                    print(f"\n[新的一天] {sim_time.strftime('%Y-%m-%d')}")

        except KeyboardInterrupt:
            print(f"\n\n[停止] 实时模拟已终止。最后模拟时间: {sim_time}")

    def run_quick(self, hours: int = 240):
        print(f"\n{'='*60}")
        print(f"[快速模式] 生成最近 {hours} 小时数据, 立即提交...")
        print(f"{'='*60}")

        start_time = datetime.now() - timedelta(hours=hours)
        current_time = start_time.replace(minute=0, second=0, microsecond=0)
        batch = []
        total_sent = 0
        start_clock = time.time()

        while current_time <= datetime.now():
            for pt in self.monitoring_points:
                reading = self.generate_single_reading(pt["id"], current_time)
                if reading:
                    batch.append(reading)

                if len(batch) >= 500:
                    self.send_batch(batch)
                    total_sent += len(batch)
                    batch = []
                    elapsed = time.time() - start_clock
                    print(f"  进度: {total_sent:,} 条, {total_sent/elapsed:.0f}条/秒")

            current_time += timedelta(hours=1)

        if batch:
            self.send_batch(batch)
            total_sent += len(batch)

        elapsed = time.time() - start_clock
        print(f"\n[完成] 快速生成完毕! 共 {total_sent:,} 条, 耗时 {elapsed:.1f}秒 ({total_sent/elapsed:.0f}条/秒)")

    def test_alert(self):
        """生成极端数据触发告警测试"""
        print("\n[告警测试] 注入异常数据以触发告警...")
        critical_points = self.monitoring_points[:3]
        batch = []
        now = datetime.now()
        for pt in critical_points:
            initial = pt.get("initialHardness", 50)
            reading = {
                "time": now.strftime("%Y-%m-%dT%H:%M:%S+08:00"),
                "pointId": pt["id"],
                "temperature": round(random.uniform(38, 45), 2),
                "humidity": round(random.uniform(92, 98), 2),
                "surfaceHardness": round(initial * 0.72, 2),
                "crackWidth": round(random.uniform(0.6, 1.2), 4),
                "windSpeed": 10.5,
                "rainfall": 0,
                "solarRadiation": 850,
                "co2Concentration": 600,
                "vibration": 0.1,
            }
            batch.append(reading)
            print(f"  注入告警数据: 点 {pt['name']} 硬度降28%, 裂隙>0.5mm")
        self.send_batch(batch)


def main():
    parser = argparse.ArgumentParser(
        description="石窟风化监测系统 - 传感器数据模拟器",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  %(prog)s quick                           # 快速生成最近10天数据
  %(prog)s historical --days 30            # 生成最近30天历史数据
  %(prog)s realtime --speed 60             # 实时模式, 每分钟模拟1小时
  %(prog)s alert                           # 触发告警测试
  %(prog)s --dry-run quick                 # 仅模拟不发送
        """,
    )
    parser.add_argument("mode", nargs="?", default="quick",
                        choices=["quick", "historical", "realtime", "alert"],
                        help="运行模式")
    parser.add_argument("--api", default="http://localhost:8080/api/v1",
                        help="后端API地址 (默认: http://localhost:8080/api/v1)")
    parser.add_argument("--days", type=int, default=365,
                        help="历史模式: 回溯天数 (默认365)")
    parser.add_argument("--hours", type=int, default=240,
                        help="快速模式: 回溯小时数 (默认240=10天)")
    parser.add_argument("--speed", type=float, default=1.0,
                        help="实时模式: 速度倍率 (默认1.0)")
    parser.add_argument("--interval", type=int, default=3600,
                        help="实时模式: 真实上报间隔秒数 (默认3600)")
    parser.add_argument("--batch-size", type=int, default=200,
                        help="批量提交大小 (默认200)")
    parser.add_argument("--dry-run", action="store_true",
                        help="仅模拟生成不发送API")
    args = parser.parse_args()

    print("=" * 60)
    print("  🏛️  古代石窟风化监测系统 - 传感器数据模拟器")
    print("=" * 60)
    print(f"  API地址: {args.api}")
    print(f"  模式: {args.mode}")
    print(f"  Dry Run: {'是' if args.dry_run else '否'}")

    sim = SensorSimulator(api_base=args.api, dry_run=args.dry_run)

    if not sim.fetch_metadata():
        print("[ERROR] 获取元数据失败，使用默认模拟数据")

    if args.mode == "quick":
        sim.run_quick(hours=args.hours)
    elif args.mode == "historical":
        sim.run_historical(days=args.days, batch_size=args.batch_size)
    elif args.mode == "realtime":
        sim.run_realtime(interval_seconds=args.interval, speed=args.speed)
    elif args.mode == "alert":
        sim.test_alert()


if __name__ == "__main__":
    main()

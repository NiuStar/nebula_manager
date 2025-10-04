#!/usr/bin/env bash
set -euo pipefail

if ! command -v ping >/dev/null 2>&1; then
  echo "[agent] 未找到 ping 命令，请先安装对应的 iputils/ping 工具" >&2
  exit 2
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "[agent] 未找到 curl 命令" >&2
  exit 2
fi

# 默认配置文件，可由安装脚本生成，NEBULA_AGENT_CONFIG 可覆盖
CONFIG_FILE="${NEBULA_AGENT_CONFIG:-/etc/nebula/nebula-network-agent.env}"

# shellcheck disable=SC1090
if [[ -f "$CONFIG_FILE" ]]; then
  # 允许配置文件覆盖环境变量
  # shellcheck source=/etc/nebula/nebula-network-agent.env
  source "$CONFIG_FILE"
fi

API_URL="${NEBULA_MANAGER_API:-http://127.0.0.1:8080}"
NODE_ID="${NEBULA_NODE_ID:-}"
PEERS_RAW="${NEBULA_PEERS:-}"
TOKEN="${NEBULA_ACCESS_TOKEN:-}"
PING_TIMEOUT="${NEBULA_AGENT_PING_TIMEOUT:-3}"
PING_COUNT="${NEBULA_AGENT_PING_COUNT:-1}"

if [[ -z "$NODE_ID" ]]; then
  echo "[agent] 需要设置 NEBULA_NODE_ID（当前节点在控制台中的 ID）" >&2
  exit 1
fi
if [[ -z "$TOKEN" ]]; then
  echo "[agent] 需要设置 NEBULA_ACCESS_TOKEN（可使用 NEBULA_STATIC_TOKEN 或登录获取）" >&2
  exit 1
fi

# 动态刷新目标列表（默认开启，可通过 NEBULA_DYNAMIC_TARGETS=0 关闭）
if [[ "${NEBULA_DYNAMIC_TARGETS:-1}" == "1" ]]; then
  TARGET_URL="$API_URL/api/nodes/${NODE_ID}/network/targets"
  if command -v python3 >/dev/null 2>&1; then
    if targets_json=$(curl -fsS -H "Authorization: Bearer ${TOKEN}" "$TARGET_URL" 2>/dev/null); then
      parsed=$(python3 - <<'PY'
import json, sys
try:
    data = json.load(sys.stdin)
except Exception:
    sys.exit(0)
targets = []
for item in data.get("data", []):
    peer_id = item.get("peer_id")
    addr = (item.get("address") or "").strip()
    if peer_id and addr:
        targets.append(f"{peer_id}:{addr}")
if targets:
    sys.stdout.write(",".join(targets))
PY
      )
      if [[ -n "$parsed" ]]; then
        PEERS_RAW="$parsed"
      fi
    else
      echo "[agent] 拉取动态目标失败：$TARGET_URL" >&2
    fi
  else
    echo "[agent] python3 不可用，跳过自动获取目标列表" >&2
  fi
fi

if [[ -z "$PEERS_RAW" ]]; then
  echo "[agent] 未配置目标节点，跳过本次上报" >&2
  exit 0
fi

IFS=',' read -r -a PEERS <<<"$PEERS_RAW"
samples="["
first=true

for entry in "${PEERS[@]}"; do
  entry_trimmed="${entry//[[:space:]]/}"
  peer_id="${entry_trimmed%%:*}"
  target="${entry_trimmed#*:}"
  if [[ -z "$peer_id" || -z "$target" || "$peer_id" == "$entry_trimmed" ]]; then
    echo "[agent] 跳过无效配置: $entry_trimmed" >&2
    continue
  fi

  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  success=true
  latency="0"

  if output=$(ping -n -c "$PING_COUNT" -W "$PING_TIMEOUT" "$target" 2>/dev/null); then
    match=$(printf '%s\n' "$output" | grep -oE 'time[=<][0-9.]+ ?ms' | tail -n 1)
    if [[ -n "$match" ]]; then
      latency=${match#time}
      latency=${latency#?}
      latency=${latency%ms*}
      latency=${latency// /}
    else
      success=false
    fi
  else
    success=false
  fi

  if [[ "$success" != true ]]; then
    latency="0"
  fi

  latency_fmt=$(printf '%.3f' "$latency")
  sample=$(printf '{"peer_id":%s,"latency_ms":%s,"success":%s,"timestamp":"%s"}' "$peer_id" "$latency_fmt" "$success" "$timestamp")

  if [[ "$first" == true ]]; then
    samples+="$sample"
    first=false
  else
    samples+=",$sample"
  fi

done

samples+=']'

payload=$(printf '{"samples":%s}' "$samples")

response=$(curl -fsS -X POST \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data "$payload" \
  "$API_URL/api/nodes/${NODE_ID}/network/samples" 2>&1) || {
    echo "[agent] 上报失败: $response" >&2
    exit 1
  }

echo "[agent] 上报完成: $response"

if [[ "${NEBULA_DISABLE_STATUS:-0}" != "1" ]]; then
  if command -v python3 >/dev/null 2>&1; then
    status_payload=$(python3 - <<'PY'
import json
import os
import shutil
import sys
import time

def read_cpu():
    def snapshot():
        with open('/proc/stat', 'r', encoding='utf-8') as f:
            parts = f.readline().split()
        values = [int(x) for x in parts[1:]]
        idle = values[3] + values[4]
        total = sum(values)
        return idle, total

    idle1, total1 = snapshot()
    time.sleep(0.2)
    idle2, total2 = snapshot()
    total_delta = total2 - total1
    if total_delta <= 0:
        return 0.0
    idle_delta = idle2 - idle1
    usage = (1 - idle_delta / total_delta) * 100.0
    if usage < 0:
        usage = 0.0
    if usage > 100:
        usage = 100.0
    return round(usage, 2)


def read_mem():
    info = {}
    with open('/proc/meminfo', 'r', encoding='utf-8') as f:
        for line in f:
            if ':' not in line:
                continue
            key, value = line.split(':', 1)
            try:
                info[key] = int(value.strip().split()[0]) * 1024
            except Exception:
                info[key] = 0
    total = info.get('MemTotal', 0)
    available = info.get('MemAvailable', 0)
    used = total - available
    swap_total = info.get('SwapTotal', 0)
    swap_free = info.get('SwapFree', 0)
    swap_used = swap_total - swap_free
    if used < 0:
        used = 0
    if swap_used < 0:
        swap_used = 0
    return total, used, swap_total, swap_used


def read_disk():
    try:
        usage = shutil.disk_usage('/')
    except Exception:
        return 0, 0
    return usage.total, usage.used


def read_net():
    rx = 0
    tx = 0
    with open('/proc/net/dev', 'r', encoding='utf-8') as f:
        for line in f:
            if ':' not in line:
                continue
            iface, data = line.split(':', 1)
            fields = data.split()
            if len(fields) < 16:
                continue
            try:
                rx += int(fields[0])
                tx += int(fields[8])
            except Exception:
                continue
    return rx, tx


def read_load():
    try:
        return os.getloadavg()
    except OSError:
        return 0.0, 0.0, 0.0


def read_uptime():
    try:
        with open('/proc/uptime', 'r', encoding='utf-8') as f:
            return int(float(f.readline().split()[0]))
    except Exception:
        return 0


def read_processes():
    try:
        with open('/proc/loadavg', 'r', encoding='utf-8') as f:
            parts = f.readline().split()
        if len(parts) >= 4 and '/' in parts[3]:
            running, total = parts[3].split('/')
            return int(total)
    except Exception:
        pass
    return 0


try:
    cpu = read_cpu()
    mem_total, mem_used, swap_total, swap_used = read_mem()
    disk_total, disk_used = read_disk()
    net_rx, net_tx = read_net()
    load1, load5, load15 = read_load()
    uptime = read_uptime()
    processes = read_processes()
    reported_at = time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())

    payload = {
        'cpu_usage': cpu,
        'load1': round(load1, 2),
        'load5': round(load5, 2),
        'load15': round(load15, 2),
        'memory_total': mem_total,
        'memory_used': mem_used,
        'swap_total': swap_total,
        'swap_used': swap_used,
        'disk_total': disk_total,
        'disk_used': disk_used,
        'net_rx_bytes': net_rx,
        'net_tx_bytes': net_tx,
        'processes': processes,
        'uptime': uptime,
        'reported_at': reported_at,
    }
    print(json.dumps(payload))
except Exception:
    pass
PY
    )
    if [[ -n "$status_payload" ]]; then
      curl -fsS -X POST \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        --data "$status_payload" \
        "$API_URL/api/nodes/${NODE_ID}/status" >/dev/null || echo "[agent] 运行状态上报失败" >&2
    fi
  else
    echo "[agent] python3 不可用，跳过运行状态上报" >&2
  fi
fi

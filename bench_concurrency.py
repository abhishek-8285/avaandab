import concurrent.futures
import time
import urllib.request
import ssl

URL = "https://avandab.com/"
NUM_USERS = 10
REQ_PER_USER = 10
TOTAL_REQUESTS = NUM_USERS * REQ_PER_USER

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

def fetch_url(request_id):
    start = time.perf_counter()
    try:
        req = urllib.request.Request(URL, headers={"User-Agent": "AvandabBench/1.0"})
        with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
            status = resp.getcode()
            elapsed = time.perf_counter() - start
            return status, elapsed
    except Exception as e:
        elapsed = time.perf_counter() - start
        return str(e), elapsed

print(f"=== Starting Concurrent Benchmark ===")
print(f"Target: {URL}")
print(f"Concurrent Users: {NUM_USERS}")
print(f"Requests per User: {REQ_PER_USER}")
print(f"Total Requests: {TOTAL_REQUESTS}\n")

overall_start = time.perf_counter()

latencies = []
statuses = {}

with concurrent.futures.ThreadPoolExecutor(max_workers=NUM_USERS) as executor:
    futures = [executor.submit(fetch_url, i) for i in range(TOTAL_REQUESTS)]
    for future in concurrent.futures.as_completed(futures):
        status, elapsed = future.result()
        latencies.append(elapsed)
        statuses[status] = statuses.get(status, 0) + 1

overall_elapsed = time.perf_counter() - overall_start

latencies.sort()
avg_latency = sum(latencies) / len(latencies)
p50 = latencies[int(len(latencies) * 0.50)]
p90 = latencies[int(len(latencies) * 0.90)]
p95 = latencies[int(len(latencies) * 0.95)]
min_lat = latencies[0]
max_lat = latencies[-1]
rps = TOTAL_REQUESTS / overall_elapsed

print(f"=== Benchmark Results ===")
print(f"Total Time Elapsed : {overall_elapsed:.3f} s")
print(f"Requests Per Sec   : {rps:.2f} req/s")
print(f"Status Codes       : {statuses}")
print(f"Min Latency        : {min_lat*1000:.2f} ms")
print(f"Average Latency    : {avg_latency*1000:.2f} ms")
print(f"P50 (Median)       : {p50*1000:.2f} ms")
print(f"P90 Latency        : {p90*1000:.2f} ms")
print(f"P95 Latency        : {p95*1000:.2f} ms")
print(f"Max Latency        : {max_lat*1000:.2f} ms")

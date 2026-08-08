import concurrent.futures
import time
import urllib.request
import ssl
import sys

URL = "https://avandab.com/"

def run_test(num_users, req_per_user):
    total_requests = num_users * req_per_user
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE

    def fetch_url(request_id):
        start = time.perf_counter()
        try:
            req = urllib.request.Request(URL, headers={"User-Agent": "AvandabMaxBench/1.0"})
            with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
                status = resp.getcode()
                elapsed = time.perf_counter() - start
                return status, elapsed
        except Exception as e:
            elapsed = time.perf_counter() - start
            return str(e), elapsed

    print(f"\n--- Testing Heavy Concurrency: {num_users} Users ({total_requests} Requests) ---")
    overall_start = time.perf_counter()
    latencies = []
    statuses = {}

    with concurrent.futures.ThreadPoolExecutor(max_workers=num_users) as executor:
        futures = [executor.submit(fetch_url, i) for i in range(total_requests)]
        for future in concurrent.futures.as_completed(futures):
            status, elapsed = future.result()
            latencies.append(elapsed)
            statuses[status] = statuses.get(status, 0) + 1

    overall_elapsed = time.perf_counter() - overall_start
    latencies.sort()
    avg_lat = (sum(latencies) / len(latencies)) * 1000
    p50 = latencies[int(len(latencies) * 0.50)] * 1000
    p95 = latencies[int(len(latencies) * 0.95)] * 1000
    rps = total_requests / overall_elapsed

    print(f"Elapsed: {overall_elapsed:.2f}s | Throughput: {rps:.2f} req/s")
    print(f"Status Codes: {statuses}")
    print(f"Avg Latency: {avg_lat:.1f} ms | P50: {p50:.1f} ms | P95: {p95:.1f} ms")
    return rps

if __name__ == "__main__":
    levels = [(1000, 5), (2000, 5), (5000, 2)]
    for users, reqs in levels:
        run_test(users, reqs)
        time.sleep(2)

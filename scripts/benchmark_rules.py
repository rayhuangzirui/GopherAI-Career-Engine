import json
import os
import statistics
import subprocess
import time
import urllib.request
import urllib.error
from typing import Dict, List, Set

API_BASE = os.getenv("API_BASE", "http://localhost:8080")
USER_ID = int(os.getenv("BENCH_USER_ID", "1"))
TOTAL_TASKS = int(os.getenv("BENCH_TOTAL_TASKS", "100"))
FAIL_EVERY = int(os.getenv("BENCH_FAIL_EVERY", "5"))
POLL_INTERVAL = float(os.getenv("BENCH_POLL_INTERVAL", "0.5"))
POLL_TIMEOUT_SECONDS = int(os.getenv("BENCH_POLL_TIMEOUT_SECONDS", "180"))

MYSQL_SERVICE = os.getenv("BENCH_MYSQL_SERVICE", "mysql")
MYSQL_USER = os.getenv("BENCH_MYSQL_USER", "app")
MYSQL_PASSWORD = os.getenv("BENCH_MYSQL_PASSWORD", "app")
MYSQL_DATABASE = os.getenv("BENCH_MYSQL_DATABASE", "appdb")

FINAL_STATUSES = {"completed", "failed", "permanently_failed"}
TASK_TYPE = "resume_jd_match"
MAX_RETRIES = 3


def http_json(method: str, path: str, payload: Dict = None) -> Dict:
    url = f"{API_BASE}{path}"
    data = None
    headers = {"Content-Type": "application/json"}

    if payload is not None:
        data = json.dumps(payload).encode("utf-8")

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            body = resp.read().decode("utf-8")
            return json.loads(body)
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")
        raise RuntimeError(f"HTTP {e.code} for {method} {path}: {body}") from e
    except urllib.error.URLError as e:
        raise RuntimeError(f"URL error for {method} {path}: {e}") from e


def percentile(values: List[float], p: float) -> float:
    if not values:
        return 0.0
    values = sorted(values)
    index = int(round((len(values) - 1) * p))
    return values[index]


def mysql_query(sql: str) -> List[str]:
    cmd = [
        "docker", "compose", "exec", "-T", MYSQL_SERVICE,
        "mysql",
        f"-u{MYSQL_USER}",
        f"-p{MYSQL_PASSWORD}",
        "-D", MYSQL_DATABASE,
        "-N", "-B",
        "-e", sql,
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"MySQL query failed:\nSTDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}")
    lines = [line.strip() for line in result.stdout.splitlines() if line.strip()]
    return lines


def fetch_processed_key_counts(task_ids: List[int]) -> Dict[int, int]:
    if not task_ids:
        return {}

    conditions = " OR ".join(
        [f"key_value LIKE '{TASK_TYPE}:{task_id}:attempt:%'" for task_id in task_ids]
    )
    sql = f"""
    SELECT key_value
    FROM processed_keys
    WHERE {conditions};
    """
    rows = mysql_query(sql)

    counts = {task_id: 0 for task_id in task_ids}
    for key in rows:
        parts = key.split(":")
        # expected: resume_jd_match:<task_id>:attempt:<n>
        if len(parts) >= 4:
            try:
                task_id = int(parts[1])
                if task_id in counts:
                    counts[task_id] += 1
            except ValueError:
                pass
    return counts


def main():
    run_id = int(time.time())
    print(f"Starting benchmark run_id={run_id}")
    print(f"API_BASE={API_BASE}, USER_ID={USER_ID}, TOTAL_TASKS={TOTAL_TASKS}, FAIL_EVERY={FAIL_EVERY}")
    print(f"Assumption: rules mode benchmark, 2 workers, MAX_RETRIES={MAX_RETRIES}\n")

    submitted = []
    create_latencies_ms = []

    # submit tasks
    for i in range(TOTAL_TASKS):
        should_fail = (FAIL_EVERY > 0 and (i + 1) % FAIL_EVERY == 0)

        resume_text = (
            f"BENCHMARK_RUN_{run_id} resume #{i+1} "
            "Go MySQL Redis RabbitMQ Docker REST APIs"
        )
        if should_fail:
            resume_text += " FAIL_ANALYSIS"

        job_description_text = (
            "Backend engineer with Go, Docker, MySQL, Redis, RabbitMQ, "
            "distributed systems, retries, idempotency, and testing."
        )

        payload = {
            "user_id": USER_ID,
            "resume_text": resume_text,
            "job_description_text": job_description_text,
        }

        t0 = time.perf_counter()
        resp = http_json("POST", "/tasks/resume-jd-match", payload)
        t1 = time.perf_counter()

        create_latency_ms = (t1 - t0) * 1000
        create_latencies_ms.append(create_latency_ms)

        submitted.append({
            "task_id": resp["task_id"],
            "should_fail": should_fail,
            "submitted_at": time.time(),
            "observed_statuses": [resp.get("status", "queued")],
        })

        print(
            f"submitted task_id={resp['task_id']} "
            f"should_fail={should_fail} create_latency_ms={create_latency_ms:.1f}"
        )

    # poll until all tasks reach final status
    pending = {item["task_id"]: item for item in submitted}
    completed_records = []

    deadline = time.time() + POLL_TIMEOUT_SECONDS

    while pending and time.time() < deadline:
        done_ids = []

        for task_id, info in pending.items():
            task = http_json("GET", f"/tasks/{task_id}")
            status = task["status"]

            if not info["observed_statuses"] or info["observed_statuses"][-1] != status:
                info["observed_statuses"].append(status)

            if status in FINAL_STATUSES:
                finished_at = time.time()
                total_latency_ms = (finished_at - info["submitted_at"]) * 1000

                record = {
                    "task_id": task_id,
                    "status": status,
                    "retry_count": task.get("retry_count", 0),
                    "should_fail": info["should_fail"],
                    "total_latency_ms": total_latency_ms,
                    "observed_statuses": info["observed_statuses"][:],
                    "result_ok": None,
                }

                if status == "completed":
                    try:
                        result_resp = http_json("GET", f"/tasks/{task_id}/result")
                        record["result_ok"] = bool(result_resp.get("ok") and result_resp.get("result"))
                    except Exception:
                        record["result_ok"] = False

                completed_records.append(record)
                done_ids.append(task_id)

        for task_id in done_ids:
            pending.pop(task_id, None)

        if pending:
            time.sleep(POLL_INTERVAL)

    if pending:
        print(f"\nWARNING: {len(pending)} tasks did not finish before timeout")
        for task_id in list(pending.keys()):
            print(f"  still pending task_id={task_id}")

    # summarize
    status_counts = {}
    retry_counts = []
    total_latencies = []
    success_latencies = []
    failed_latencies = []

    validation_errors = []

    for record in completed_records:
        status = record["status"]
        status_counts[status] = status_counts.get(status, 0) + 1
        retry_counts.append(record["retry_count"])
        total_latencies.append(record["total_latency_ms"])

        if status == "completed":
            success_latencies.append(record["total_latency_ms"])
        else:
            failed_latencies.append(record["total_latency_ms"])

        # Validate expected status behavior
        if record["should_fail"]:
            if status != "permanently_failed":
                validation_errors.append(
                    f"task {record['task_id']} expected permanently_failed but got {status}"
                )
            if record["retry_count"] != MAX_RETRIES:
                validation_errors.append(
                    f"task {record['task_id']} expected retry_count={MAX_RETRIES} but got {record['retry_count']}"
                )
        else:
            if status != "completed":
                validation_errors.append(
                    f"task {record['task_id']} expected completed but got {status}"
                )
            if record["retry_count"] != 0:
                validation_errors.append(
                    f"task {record['task_id']} expected retry_count=0 but got {record['retry_count']}"
                )
            if record["result_ok"] is not True:
                validation_errors.append(
                    f"task {record['task_id']} completed but /result was not readable"
                )

    # DB verification for processed_keys
    task_ids = [r["task_id"] for r in completed_records]
    processed_key_counts = {}
    db_check_ok = True
    db_check_errors = []

    try:
        processed_key_counts = fetch_processed_key_counts(task_ids)

        for record in completed_records:
            task_id = record["task_id"]
            actual = processed_key_counts.get(task_id, 0)
            expected = 1 if record["status"] == "completed" else (MAX_RETRIES + 1)

            if actual != expected:
                db_check_ok = False
                db_check_errors.append(
                    f"task {task_id} expected {expected} processed_keys but found {actual}"
                )
    except Exception as e:
        db_check_ok = False
        db_check_errors.append(f"processed_keys verification failed: {e}")

    # print summary
    print("\n=== ENHANCED BENCHMARK SUMMARY ===")
    print(f"run_id: {run_id}")
    print(f"submitted tasks: {len(submitted)}")
    print(f"finished tasks: {len(completed_records)}")
    print(f"unfinished tasks: {len(pending)}")
    print(f"status counts: {status_counts}")

    if create_latencies_ms:
        print("\nCreate latency (ms)")
        print(f"  avg: {statistics.mean(create_latencies_ms):.1f}")
        print(f"  p50: {percentile(create_latencies_ms, 0.50):.1f}")
        print(f"  p95: {percentile(create_latencies_ms, 0.95):.1f}")

    if total_latencies:
        print("\nCompletion latency (ms)")
        print(f"  avg: {statistics.mean(total_latencies):.1f}")
        print(f"  p50: {percentile(total_latencies, 0.50):.1f}")
        print(f"  p95: {percentile(total_latencies, 0.95):.1f}")

    if success_latencies:
        print("\nSuccess latency (ms)")
        print(f"  avg: {statistics.mean(success_latencies):.1f}")
        print(f"  p50: {percentile(success_latencies, 0.50):.1f}")
        print(f"  p95: {percentile(success_latencies, 0.95):.1f}")

    if failed_latencies:
        print("\nFailure latency (ms)")
        print(f"  avg: {statistics.mean(failed_latencies):.1f}")
        print(f"  p50: {percentile(failed_latencies, 0.50):.1f}")
        print(f"  p95: {percentile(failed_latencies, 0.95):.1f}")

    if retry_counts:
        print("\nRetry count")
        print(f"  avg: {statistics.mean(retry_counts):.2f}")
        print(f"  max: {max(retry_counts)}")

    print("\nPer-task status traces")
    for record in completed_records:
        print(
            f"  task {record['task_id']}: "
            f"{' -> '.join(record['observed_statuses'])} "
            f"(final={record['status']}, retries={record['retry_count']})"
        )

    print("\nProcessed key counts")
    for record in completed_records:
        task_id = record["task_id"]
        expected = 1 if record["status"] == "completed" else (MAX_RETRIES + 1)
        actual = processed_key_counts.get(task_id, "N/A")
        print(f"  task {task_id}: expected={expected}, actual={actual}")

    print("\nValidation")
    if validation_errors:
        print("  task behavior validation: FAILED")
        for err in validation_errors:
            print(f"   - {err}")
    else:
        print("  task behavior validation: PASSED")

    if db_check_ok:
        print("  processed_keys / idempotency validation: PASSED")
    else:
        print("  processed_keys / idempotency validation: FAILED")
        for err in db_check_errors:
            print(f"   - {err}")

    no_duplicate_finalization_observed = (not validation_errors) and db_check_ok and not pending
    print(f"\nNo duplicate finalization observed: {no_duplicate_finalization_observed}")

    print("\nExpected fail count (approx):", TOTAL_TASKS // FAIL_EVERY if FAIL_EVERY > 0 else 0)
    print("Done.")


if __name__ == "__main__":
    main()
"use client";

import { useEffect, useRef, useState } from "react";
import { AnalysisForm } from "@/components/analysis-form";
import { HistoryList } from "@/components/history-list";
import { ResultReport } from "@/components/result-report";
import { StatusCard } from "@/components/status-card";
import {
  createResumeJDMatchTask,
  getTask,
  getTaskHistory,
  getTaskResult,
} from "@/lib/api";
import {
  ResumeJDMatchResult,
  TaskStatus,
  TaskSummary,
} from "@/lib/types";

const FINAL_STATUSES: TaskStatus[] = ["completed", "failed", "permanently_failed"];
type UiPhase = "preparing" | "comparing" | "generating" | null;

export default function Page() {
  const [taskId, setTaskId] = useState<number | null>(null);
  const [status, setStatus] = useState<TaskStatus | null>(null);
  const [retryCount, setRetryCount] = useState<number>(0);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [result, setResult] = useState<ResumeJDMatchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [history, setHistory] = useState<TaskSummary[]>([]);

  const [uiPhase, setUiPhase] = useState<UiPhase>(null);

  const pollingRef = useRef<number | null>(null);
  const phaseTimeoutsRef = useRef<number[]>([]);

  function stopPolling() {
    if (pollingRef.current !== null) {
      window.clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }

  function clearPhaseTimers() {
    phaseTimeoutsRef.current.forEach((id) => window.clearTimeout(id));
    phaseTimeoutsRef.current = [];
  }

  function scheduleUiPhases() {
    clearPhaseTimers();
    setUiPhase("preparing");

    const comparingTimer = window.setTimeout(() => {
      setUiPhase((current) => {
        if (current === "preparing") return "comparing";
        return current;
      });
    }, 700);

    const generatingTimer = window.setTimeout(() => {
      setUiPhase((current) => {
        if (current === "preparing" || current === "comparing") return "generating";
        return current;
      });
    }, 1800);

    phaseTimeoutsRef.current = [comparingTimer, generatingTimer];
  }

  async function refreshHistory() {
    try {
      const data = await getTaskHistory(10);
      setHistory(data.tasks ?? []);
    } catch (error) {
      console.error("Failed to fetch task history:", error);
    }
  }

  function startPolling(id: number) {
    stopPolling();

    pollingRef.current = window.setInterval(async () => {
      try {
        const task = await getTask(id);
        setStatus(task.status);
        setRetryCount(task.retry_count);
        setErrorMessage(task.error_message);

        if (FINAL_STATUSES.includes(task.status)) {
          stopPolling();
          clearPhaseTimers();
          setUiPhase(null);

          if (task.status === "completed") {
            const taskResult = await getTaskResult(id);
            if (taskResult.result) {
              setResult(taskResult.result);
            }
          }

          await refreshHistory();
          setLoading(false);
        }
      } catch (error) {
        console.error("Failed to fetch task status:", error);
        stopPolling();
        clearPhaseTimers();
        setUiPhase(null);
        setLoading(false);
      }
    }, 500);
  }

  async function handleSubmit(payload: {
    resumeText: string;
    jobDescriptionText: string;
  }) {
    setLoading(true);
    setResult(null);
    setErrorMessage(null);
    setUiPhase("preparing");
    scheduleUiPhases();

    try {
      const created = await createResumeJDMatchTask(payload);
      console.log("created task response object:", created);

      setTaskId(created.task_id);
      setStatus(created.status);
      setRetryCount(0);

      startPolling(created.task_id);
    } catch (error) {
      console.error("Failed to create task:", error);
      stopPolling();
      clearPhaseTimers();
      setUiPhase(null);
      setLoading(false);
    }
  }

  async function loadTaskFromHistory(selectedTaskId: number) {
    stopPolling();
    clearPhaseTimers();
    setUiPhase(null);
    setTaskId(selectedTaskId);
    setResult(null);
    setLoading(false);

    try {
      const task = await getTask(selectedTaskId);
      setStatus(task.status);
      setRetryCount(task.retry_count);
      setErrorMessage(task.error_message);

      if (task.status === "completed") {
        const taskResult = await getTaskResult(selectedTaskId);
        if (taskResult.result) {
          setResult(taskResult.result);
        }
      } else if (!FINAL_STATUSES.includes(task.status)) {
        setUiPhase("preparing");
        scheduleUiPhases();
        startPolling(selectedTaskId);
      }
    } catch (error) {
      console.error("Failed to load task:", error);
    }
  }

  useEffect(() => {
    refreshHistory();

    return () => {
      stopPolling();
      clearPhaseTimers();
    };
  }, []);

  return (
      <main className="min-h-screen bg-gray-50 px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8">
            <div className="text-sm font-medium text-gray-500">Demo workspace</div>
            <h1 className="mt-2 text-3xl font-bold text-gray-900">
              Optimize your resume for a specific job
            </h1>
            <p className="mt-3 max-w-3xl text-sm text-gray-600">
              Submit your resume and a target job description to get a user-friendly
              match report, keyword gaps, and prioritized improvement.
            </p>
          </div>

          <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
            <div className="space-y-6">
              <AnalysisForm onSubmit={handleSubmit} loading={loading} />
              <HistoryList tasks={history} onSelect={loadTaskFromHistory} />
            </div>

            <div className="space-y-6">
              {status && (
                  <StatusCard
                      status={status}
                      retryCount={retryCount}
                      errorMessage={errorMessage}
                      uiPhase={uiPhase}
                  />
              )}

              {taskId && (
                  <div className="rounded-2xl border bg-white p-4 text-sm text-gray-600 shadow-sm">
                    Current task ID:{" "}
                    <span className="font-medium text-gray-900">#{taskId}</span>
                  </div>
              )}

              {result && <ResultReport result={result} />}
            </div>
          </div>
        </div>
      </main>
  );
}
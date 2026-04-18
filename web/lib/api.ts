import {
    CreateTaskResponse,
    TaskDetailResponse,
    TaskListResponse,
    TaskResultResponse,
} from "@/lib/types";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";
const DEMO_USER_ID = 1;

export async function createResumeJDMatchTask(payload: {
    resumeText: string;
    jobDescriptionText: string;
}): Promise<CreateTaskResponse> {
    const requestBody = {
        "user_id": DEMO_USER_ID,
        "resume_text": payload.resumeText,
        "job_description_text": payload.jobDescriptionText,
    };

    console.log("Sending request to API:", requestBody);

    const response = await fetch(`${API_BASE}/tasks/resume-jd-match`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to create task: ${response.status} ${errorText}`);
    }

    return response.json();
}

export async function getTask(taskId: number): Promise<TaskDetailResponse> {
    const response = await fetch(`${API_BASE}/tasks/${taskId}`, {
        cache: "no-store"
    });

    if (!response.ok) {
        throw new Error(`Failed to fetch task: ${response.status}`);
    }

    return response.json();
}

export async function getTaskResult(taskId: number): Promise<TaskResultResponse> {
    const res = await fetch(`${API_BASE}/tasks/${taskId}/result`, {
       cache: "no-store",
    });

    if (!res.ok) {
        throw new Error(`Failed to fetch task result: ${res.status}`);
    }

    return res.json();
}

export async function getTaskHistory(limit = 10): Promise<TaskListResponse> {
    const res = await fetch(`${API_BASE}/tasks?user_id=${DEMO_USER_ID}&limit=${limit}`,
    { cache: "no-store" },
    );

    if (!res.ok) {
        throw new Error(`Failed to fetch task history: ${res.status}`)
    }

    return res.json();
}
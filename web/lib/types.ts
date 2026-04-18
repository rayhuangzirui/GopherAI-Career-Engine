export type TaskStatus =
    | "pending"
    | "queued"
    | "processing"
    | "retrying"
    | "completed"
    | "failed"
    | "permanently_failed";

export type TaskSummary = {
    id: number;
    user_id: number;
    task_type: string;
    status: TaskStatus;
    retry_count: number;
    error_message: string | null;
    started_at: string | null;
    completed_at: string | null;
    created_at: string;
    updated_at: string;
};

export type ResumeJDMatchResult = {
    match_score: number;
    matched_keywords: string[];
    missing_keywords: string[];
    suggestions: string[];
    source?: string;
    semantic_alignment_summary?: string;
};

export type CreateTaskResponse = {
    ok: boolean;
    task_id: number;
    status: TaskStatus;
};

export type TaskDetailResponse = TaskSummary & {
    ok: boolean;
};

export type TaskResultResponse = {
    ok: boolean;
    status: TaskStatus;
    result?: ResumeJDMatchResult;
    error_message?: string;
    message?: string;
}

export type TaskListResponse = {
    ok: boolean;
    tasks: TaskSummary[];
};
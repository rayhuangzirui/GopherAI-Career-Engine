import { TaskStatus } from "@/lib/types";

type UiPhase = "preparing" | "comparing" | "generating" | null;

function getDisplayLabel(status: TaskStatus, uiPhase: UiPhase) {
    if (status === "completed") return "Your match report is ready";
    if (status === "failed" || status === "permanently_failed") {
        return "We couldn't finish this analysis";
    }
    if (status === "retrying") return "Retrying after a temporary issue";

    if (uiPhase === "generating") return "Generating recommendations";
    if (uiPhase === "comparing") return "Comparing your resume to the job description";
    return "Preparing your analysis";
}

export function StatusCard({
                               status,
                               retryCount,
                               errorMessage,
                               uiPhase,
                           }: {
    status: TaskStatus;
    retryCount?: number;
    errorMessage?: string | null;
    uiPhase?: UiPhase;
}) {
    return (
        <div className="rounded-2xl border bg-white p-5 shadow-sm">
            <div className="mb-2 text-sm font-medium text-gray-500">Current status</div>
            <div className="text-xl font-semibold text-gray-900">
                {getDisplayLabel(status, uiPhase ?? null)}
            </div>

            <div className="mt-3 text-sm text-gray-600">
                Internal status: <span className="font-medium">{status}</span>
            </div>

            {typeof retryCount === "number" && retryCount > 0 && (
                <div className="mt-2 text-sm text-gray-600">
                    Retries so far: <span className="font-medium">{retryCount}</span>
                </div>
            )}

            {errorMessage && (
                <div className="mt-4 rounded-xl bg-red-50 p-3 text-sm text-red-700">
                    {errorMessage}
                </div>
            )}
        </div>
    );
}
import { TaskSummary } from "@/lib/types";

export function HistoryList({
    tasks,
    onSelect,
}: {
    tasks: TaskSummary[];
    onSelect: (taskId: number) => void;
}) {
    return (
        <div className="rounded-2xl border bg-white p-5 shadow-md">
            <div className="mb-4 text-sm font-medium text-gray-500">Recent analyses</div>

            <div className="space-y-3">
                {tasks.length === 0 ? (
                    <p className="text-sm text-gray-500">No analysis history.</p>
                ) : (
                    tasks.map((task) => (
                        <button
                            key={task.id}
                            className="w-full rounded-xl border p-4 text-left transition hover:bg-gray-50"
                            onClick={() => onSelect(task.id)}
                        >
                            <div className="flex items-center justify-between">
                                <div className="text-sm font-semibold text-gray-900">
                                    Task #{task.id}
                                </div>
                                <div className="text-xs text-gray-500">{task.status}</div>
                            </div>
                            <div className="mt-2 text-xs text-gray-500">
                                {new Date(task.created_at).toLocaleString()}
                            </div>
                            </button>
                    ))
                )}
            </div>
        </div>
    );
}
import { ResumeJDMatchResult } from "@/lib/types";

export function ResultReport({ result }: { result: ResumeJDMatchResult }) {
    return (
        <div className="space-y-6 rounded-2xl border bg-white p-6 shadow-sm">
            <section>
                <div className="mb-2 text-sm font-medium text-gray-500">Match overview</div>
                <div className="flex items-end gap-3">
                    <div className="text-4xl font-bold text-gray-900">{result.match_score}</div>
                    <div className="pb-1 text-sm text-gray-500">/ 100</div>
                </div>
                <p className="mt-2 text-sm text-gray-600">
                    This score is a resume-to-job relevance estimate for optimization and comparison,
                    not a hiring prediction.
                </p>
                {result.source && (
                    <div className="mt-3 inline-flex rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700">
                        Source: {result.source}
                    </div>
                )}
            </section>

            <section>
                <div className="mb-3 text-sm font-medium text-gray-500">Strong matches</div>
                <div className="flex flex-wrap gap-2">
                    {result.matched_keywords.length > 0 ? (
                        result.matched_keywords.map((item) => (
                            <span
                                key={item}
                                className="rounded-full bg-green-50 px-3 py-1 text-sm text-green-700"
                            >
                {item}
              </span>
                        ))
                    ) : (
                        <p className="text-sm text-gray-500">No strong matches detected.</p>
                    )}
                </div>
            </section>

            <section>
                <div className="mb-3 text-sm font-medium text-gray-500">Gaps to address</div>
                <div className="flex flex-wrap gap-2">
                    {result.missing_keywords.length > 0 ? (
                        result.missing_keywords.map((item) => (
                            <span
                                key={item}
                                className="rounded-full bg-amber-50 px-3 py-1 text-sm text-amber-700"
                            >
                {item}
              </span>
                        ))
                    ) : (
                        <p className="text-sm text-gray-500">No major gaps detected.</p>
                    )}
                </div>
            </section>

            <section>
                <div className="mb-3 text-sm font-medium text-gray-500">Top recommended changes</div>
                <div className="space-y-3">
                    {result.suggestions.length > 0 ? (
                        result.suggestions.map((item, idx) => (
                            <div key={`${idx}-${item}`} className="rounded-xl bg-gray-50 p-4 text-sm text-gray-700">
                                {item}
                            </div>
                        ))
                    ) : (
                        <p className="text-sm text-gray-500">No suggestions available.</p>
                    )}
                </div>
            </section>
        </div>
    );
}
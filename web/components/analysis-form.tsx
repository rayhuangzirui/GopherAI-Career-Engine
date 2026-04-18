"use client"

import { useState } from "react";

export function AnalysisForm({
    onSubmit,
    loading,
}: {
    onSubmit: (payload: {resumeText: string; jobDescriptionText: string}) => Promise<void>;
    loading: boolean;
}) {
    const [resumeText, setResumeText] = useState("");
    const [jobDescriptionText, setJobDescriptionText] = useState("");

    return (
        <form
            className="space-y-6 rounded-2xl border bg-white p-6 shadow-sm"
            onSubmit={async (e) => {
                e.preventDefault();
                console.log("Submitting form: ", { resumeText, jobDescriptionText });
                await onSubmit({ resumeText, jobDescriptionText });
            }}
        >
            <section>
                <div className="mb-2 text-lg font-semibold text-gray-900">Target job</div>
                <p className="mb-3 text-sm text-gray-600">
                    Paste the job description you want to optimize for.
                </p>
                <textarea
                    value={jobDescriptionText}
                    onChange={(e) => setJobDescriptionText(e.target.value)}
                    rows={10}
                    className="w-full rounded-xl border p-3 text-sm outline-none focus:ring-2 focus:ring-black/10"
                    placeholder="Paste your job description here..."
                    required
                />
            </section>

            <section>
                <div className="mb-2 text-lg font-semibold text-gray-900">Your resume</div>
                <p className="mb-3 text-sm text-gray-600">
                    Paste your resume in plain text.
                </p>
                <textarea
                    value={resumeText}
                    onChange={(e) => setResumeText(e.target.value)}
                    rows={12}
                    className="w-full rounded-xl border p-3 text-sm outline-none focus:ring-2 focus:ring-black/10"
                    placeholder="Paste your resume here..."
                    required
                />
            </section>

            <button
                type="submit"
                disabled={loading}
                className="rounded-xl bg-black px-5 py-3 text-sm font-medium text-white transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
            >
                {loading ? "Stating analysis..." : "Get Resume Match Report"}
            </button>
        </form>
    )
}
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSocket } from '../utils/Socket';

export default function Home() {
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();
    const { sendMessage, subscribe } = useSocket();

    const handleFormSubmit = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const formData = new FormData(event.currentTarget);
        const projectType = formData.get("option") as string;

        if (!projectType) return;

        setLoading(true);
        
        // Subscribe to success response
        const unsubscribe = subscribe("Project created successfully", (payload: any) => {
            setLoading(false);
            unsubscribe();
            console.log(payload)
            navigate(`/workspace/${payload.workspaceId}`, { state: { tree: payload.fileNode }});
        });

        const typeMap: Record<string, string> = {
            "1": "cpp", "2": "python", "3": "golang", "4": "nodejs", "5":"react"
        };
        
        sendMessage("initProject", {
            projectType: typeMap[projectType] || "python"
        });
    };

    return (
        <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center text-slate-200">
            <h1 className="text-4xl font-bold mb-8 text-blue-500">CloudIDE</h1>
            <div className="p-8 bg-slate-900 border border-slate-700 rounded-xl shadow-2xl w-full max-w-md">
                <form className="space-y-6" onSubmit={handleFormSubmit}>
                    <h2 className="text-xl font-semibold mb-4">Select Template</h2>
                    <div className="grid grid-cols-2 gap-4">
                        {["C++", "Python", "Golang", "NodeJS", "React"].map((lang, idx) => (
                            <label key={lang} className="cursor-pointer">
                                <input type="radio" name="option" value={idx + 1} className="peer sr-only" />
                                <div className="p-4 rounded-lg border border-slate-700 bg-slate-800 hover:bg-slate-700 peer-checked:border-blue-500 peer-checked:bg-blue-500/10 transition-all text-center">
                                    {lang}
                                </div>
                            </label>
                        ))}
                    </div>
                    <button 
                        type="submit" 
                        disabled={loading}
                        className="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white font-bold rounded-lg transition-all disabled:opacity-50"
                    >
                        {loading ? "Creating Environment..." : "Create Workspace"}
                    </button>
                </form>
            </div>
        </div>
    );
}
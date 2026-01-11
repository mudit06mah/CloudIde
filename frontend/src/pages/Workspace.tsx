import { useState } from "react";
import { useParams, useLocation } from "react-router-dom";
import { useSocket } from "../utils/Socket";
import Folder from "../components/tree/Folder";
import CodeEditor from "../components/editor/CodeEditor"; // Ensure casing matches file system
import Xterm from "../components/terminal/Xterm";

interface FileNode {
    name: string;
    type: string;
    children: FileNode[];
    path: string;
}

export default function Workspace() {
    const { workspaceId } = useParams();
    const location = useLocation();
    const { sendMessage, subscribe } = useSocket();
    
    // We use a safe cast or default to null. 
    // In a real app, you might fetch this if location.state is missing.
    const fileTree = (location.state?.tree as FileNode) || null;
    
    const [selectedFile, setSelectedFile] = useState<{ path: string, content: string } | null>(null);

    const handleFileSelect = (path: string) => {
        // Subscribe to the response
        const unsubscribe = subscribe("File retrieved successfully", (payload: any) => {
            setSelectedFile({ path, content: payload.content });
            unsubscribe();
        });

        // Backend expects 'filePath' to be the directory and 'fileName' to be the name.
        // We need to calculate this relative to the project root.
        
        let relativePath = path;
        if (fileTree && fileTree.path && path.startsWith(fileTree.path)) {
            relativePath = path.substring(fileTree.path.length).replace(/^\//, "");
        }

        const fileName = relativePath.split("/").pop() || "";
        const fileDir = relativePath.substring(0, relativePath.lastIndexOf("/"));

        sendMessage("getFile", {
            fileName: fileName,
            filePath: fileDir 
        });
    };

    return (
        <div className="h-screen flex flex-col bg-slate-950 text-slate-300 overflow-hidden">
            <div className="h-12 border-b border-slate-800 flex items-center px-4 bg-slate-900">
                <span className="font-mono text-sm text-blue-400">workspace: {workspaceId}</span>
            </div>
            
            <div className="flex-1 flex overflow-hidden">
                {/* File Explorer */}
                <div className="w-64 bg-slate-900 border-r border-slate-800 flex flex-col">
                    <div className="p-3 text-xs font-bold text-slate-500 uppercase tracking-wider">Explorer</div>
                    <div className="flex-1 overflow-y-auto p-2">
                        {fileTree ? (
                            <Folder Node={fileTree} onSelectFile={handleFileSelect} />
                        ) : (
                            <div className="text-slate-600 text-sm p-2">Loading tree...</div>
                        )}
                    </div>
                </div>

                {/* Main Content */}
                <div className="flex-1 flex flex-col min-w-0">
                    {/* Editor */}
                    <div className="flex-1 relative bg-slate-950">
                        {selectedFile ? (
                            <CodeEditor 
                                initialContent={selectedFile.content} 
                                filePath={selectedFile.path}
                                rootPath={fileTree?.path} 
                            />
                        ) : (
                            <div className="h-full flex items-center justify-center text-slate-600">
                                <div className="text-center">
                                    <p className="mb-2">Select a file to edit</p>
                                    <p className="text-xs text-slate-700">Changes are saved automatically</p>
                                </div>
                            </div>
                        )}
                    </div>
                    
                    {/* Terminal */}
                    <div className="h-48 border-t border-slate-800 bg-black">
                        {workspaceId && <Xterm workspaceId={workspaceId} />}
                    </div>
                </div>
            </div>
        </div>
    );
}
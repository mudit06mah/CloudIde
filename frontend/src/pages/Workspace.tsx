import { useState, useEffect } from "react";
import { useParams, useLocation } from "react-router-dom";
import { useSocket } from "../utils/Socket";
import Folder from "../components/tree/Folder";
import CodeEditor from "../components/editor/CodeEditor";
import Xterm from "../components/terminal/Xterm";
import { VscNewFile, VscNewFolder, VscRefresh } from "react-icons/vsc";
import { FaExternalLinkAlt } from "react-icons/fa";

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
    
    // Initial state from navigation, fallback to null
    const [fileTree, setFileTree] = useState<FileNode | null>(
        (location.state?.tree as FileNode) || null
    );
    
    const [selectedFile, setSelectedFile] = useState<{ path: string, content: string } | null>(null);
    const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
    const [creatingConfig, setCreatingConfig] = useState<{ parentPath: string; type: "file" | "folder" } | null>(null);

    // Dynamic Preview URL (using nip.io for domain-less setup)
    const previewUrl = `http://${workspaceId}-preview.127.0.0.1.nip.io`;

    // 1. Check if this is a React Project based on file structure
    // We look for 'index.html' or 'vite.config.js' in the root
    const isReactProject = fileTree?.children?.some(child => 
        child.name === "index.html" || 
        child.name === "vite.config.js" || 
        child.name === "vite.config.ts"
    );

    useEffect(() => {
        if (!fileTree && workspaceId) {
            sendMessage("getTree", { workspaceId });
        }
    }, [fileTree, workspaceId, sendMessage]);

    useEffect(() => {
        const unsubscribe = subscribe("Succesfully generated tree", (payload: any) => {
            if (payload.tree) {
                setFileTree(payload.tree);
                if (!selectedFolder) {
                    setSelectedFolder(payload.tree.path);
                }
            }
        });
        return unsubscribe;
    }, [subscribe, selectedFolder]);

    const handleNodeSelect = (node: FileNode) => {
        if (node.type === "file") {
            const unsubscribe = subscribe("File retrieved successfully", (payload: any) => {
                setSelectedFile({ path: node.path, content: payload.content });
                unsubscribe();
            });
            sendMessage("getFile", { filePath: node.path });
        } else {
            setSelectedFolder(node.path);
        }
    };

    const initiateCreate = (type: "file" | "folder") => {
        if (!selectedFolder) return;
        setCreatingConfig({ parentPath: selectedFolder, type });
    };

    const handleCreateSubmit = (name: string, parentPath: string, type: "file" | "folder") => {
        const messageType = type === "file" ? "createFile" : "createFolder";
        const payload = type === "file" 
            ? { fileName: name, filePath: parentPath } 
            : { folderName: name, folderPath: parentPath };

        sendMessage(messageType, payload);
        setCreatingConfig(null);
        setTimeout(() => sendMessage("getTree", { workspaceId }), 200);
    };

    const handleDeleteNode = (path: string, type: "file" | "folder", name: string) => {
        const messageType = type === "file" ? "deleteFile" : "deleteFolder";
        const dirPath = path.substring(0, path.lastIndexOf("/"));
        
        const payload = type === "file" 
            ? { fileName: name, filePath: dirPath }
            : { folderPath: path }; 

        sendMessage(messageType, payload);
        setTimeout(() => sendMessage("getTree", { workspaceId }), 200);
    };

    return (
        <div className="h-screen flex flex-col bg-slate-950 text-slate-300 overflow-hidden">
            {/* TOP HEADER */}
            <div className="h-10 border-b border-slate-800 flex items-center px-4 bg-slate-900 justify-between">
                <div className="flex items-center gap-2">
                    <span className="font-mono text-sm text-slate-400">workspace: <span className="text-blue-400">{workspaceId}</span></span>
                </div>

                {isReactProject && (
                    <a 
                        href={previewUrl} 
                        target="_blank" 
                        rel="noreferrer"
                        className="bg-blue-600 hover:bg-blue-500 text-white text-xs px-3 py-1.5 rounded flex items-center gap-2 transition-colors font-medium"
                        title="Open Preview"
                    >
                        <span>Preview</span>
                        <FaExternalLinkAlt size={10} />
                    </a>
                )}
            </div>
            
            <div className="flex-1 flex overflow-hidden">
                {/* File Explorer */}
                <div className="w-64 bg-slate-900 border-r border-slate-800 flex flex-col">
                    <div className="p-3 flex items-center justify-between text-xs font-bold text-slate-500 uppercase tracking-wider bg-slate-900 sticky top-0 z-10">
                        <span>Explorer</span>
                        <div className="flex gap-3 text-slate-400">
                            <button onClick={() => initiateCreate("file")} className="hover:text-blue-400 transition-colors" title="New File">
                                <VscNewFile size={16} />
                            </button>
                            <button onClick={() => initiateCreate("folder")} className="hover:text-blue-400 transition-colors" title="New Folder">
                                <VscNewFolder size={16} />
                            </button>
                            <button onClick={() => sendMessage("getTree", { workspaceId })} className="hover:text-green-400 transition-colors" title="Refresh">
                                <VscRefresh size={16} />
                            </button>
                        </div>
                    </div>

                    <div className="flex-1 overflow-y-auto p-2">
                        {fileTree ? (
                            <Folder 
                                node={fileTree} 
                                selectedFolder={selectedFolder}
                                creatingConfig={creatingConfig}
                                onSelect={handleNodeSelect}
                                onCreateSubmit={handleCreateSubmit}
                                onCancelCreate={() => setCreatingConfig(null)}
                                onDelete={handleDeleteNode}
                            />
                        ) : (
                            <div className="text-slate-600 text-sm p-2 text-center mt-4">Loading tree...</div>
                        )}
                    </div>
                </div>

                {/* Main Content */}
                <div className="flex-1 flex flex-col min-w-0">
                    <div className="flex-1 relative bg-slate-950">
                        {selectedFile ? (
                            <CodeEditor 
                                initialContent={selectedFile.content} 
                                filePath={selectedFile.path} 
                            />
                        ) : (
                            <div className="h-full flex items-center justify-center text-slate-600">
                                <p>Select a file to edit</p>
                            </div>
                        )}
                    </div>
                    
                    <div className="h-48 border-t border-slate-800 bg-black">
                        {workspaceId && <Xterm workspaceId={workspaceId} />}
                    </div>
                </div>
            </div>
        </div>
    );
}
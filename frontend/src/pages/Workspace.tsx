import { useState, useEffect } from "react";
import { useParams, useLocation } from "react-router-dom";
import { useSocket } from "../utils/Socket";
import Folder from "../components/tree/Folder";
import CodeEditor from "../components/editor/CodeEditor";
import Xterm from "../components/terminal/Xterm";
import { VscNewFile, VscNewFolder, VscRefresh } from "react-icons/vsc";

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
    
    // Creating State: Where and what are we creating?
    const [creatingConfig, setCreatingConfig] = useState<{ parentPath: string; type: "file" | "folder" } | null>(null);

    // 1. Handle Refresh: Fetch tree if missing
    useEffect(() => {
        if (!fileTree && workspaceId) {
            sendMessage("getTree", { workspaceId });
        }
    }, [fileTree, workspaceId, sendMessage]);

    // 2. Listen for Tree Updates (Refreshes file explorer)
    useEffect(() => {
        const unsubscribe = subscribe("Succesfully generated tree", (payload: any) => {
            console.log(payload)
            if (payload.tree) {
                setFileTree(payload.tree);
                // Default selection to root if nothing selected
                if (!selectedFolder) {
                    setSelectedFolder(payload.tree.path);
                }
            }
        });
        return unsubscribe;
    }, [subscribe, selectedFolder]);

    const handleNodeSelect = (node: FileNode) => {
        if (node.type === "file") {
            // Fetch File Content
            const unsubscribe = subscribe("File retrieved successfully", (payload: any) => {
                setSelectedFile({ path: node.path, content: payload.content });
                unsubscribe();
            });
            sendMessage("getFile", { filePath: node.path }); // Using path directly
        } else {
            // Just select the folder for potential creation context
            setSelectedFolder(node.path);
        }
    };

    const initiateCreate = (type: "file" | "folder") => {
        if (!selectedFolder) return; // Should visually disable buttons if no folder selected
        setCreatingConfig({ parentPath: selectedFolder, type });
    };

    const handleCreateSubmit = (name: string, parentPath: string, type: "file" | "folder") => {
        const messageType = type === "file" ? "createFile" : "createFolder";
        // Payload using the path directly from the node
        const payload = type === "file" 
            ? { fileName: name, filePath: parentPath } 
            : { folderName: name, folderPath: parentPath };
        sendMessage(messageType, payload);
        
        // Cleanup UI
        setCreatingConfig(null);
        
        // Refresh Tree
        // Note: You might want to listen to "File created successfully" to trigger this,
        // but a quick timeout works for immediate feedback in prototypes.
        setTimeout(() => {
            sendMessage("getTree", { workspaceId });
        }, 200);
    };

    const handleDeleteNode = (path: string, type: "file" | "folder", name: string) => {
        const messageType = type === "file" ? "deleteFile" : "deleteFolder";
        // Use path directly. For backend compatibility, we might need to split 
        // if backend expects 'path' AND 'name' separately. 
        // Based on your handlers.go: 
        // deleteFile expects { fileName, filePath }
        // deleteFolder expects { folderPath } which seems to be the full path?
        
        // Let's derive directory for files to match backend expectation
        const dirPath = path.substring(0, path.lastIndexOf("/"));
        
        const payload = type === "file" 
            ? { fileName: name, filePath: dirPath }
            : { folderPath: path }; 

        sendMessage(messageType, payload);

        setTimeout(() => {
            sendMessage("getTree", { workspaceId });
        }, 200);
    };

    return (
        <div className="h-screen flex flex-col bg-slate-950 text-slate-300 overflow-hidden">
            <div className="h-10 border-b border-slate-800 flex items-center px-4 bg-slate-900 justify-between">
                <span className="font-mono text-sm text-slate-400">workspace: <span className="text-blue-400">{workspaceId}</span></span>
            </div>
            
            <div className="flex-1 flex overflow-hidden">
                {/* File Explorer */}
                <div className="w-64 bg-slate-900 border-r border-slate-800 flex flex-col">
                    <div className="p-3 flex items-center justify-between text-xs font-bold text-slate-500 uppercase tracking-wider bg-slate-900 sticky top-0 z-10">
                        <span>Explorer</span>
                        <div className="flex gap-3 text-slate-400">
                            <button 
                                onClick={() => initiateCreate("file")} 
                                className="hover:text-blue-400 transition-colors"
                                title="New File"
                            >
                                <VscNewFile size={16} />
                            </button>
                            <button 
                                onClick={() => initiateCreate("folder")} 
                                className="hover:text-blue-400 transition-colors"
                                title="New Folder"
                            >
                                <VscNewFolder size={16} />
                            </button>
                            <button 
                                onClick={() => sendMessage("getTree", { workspaceId })} 
                                className="hover:text-green-400 transition-colors"
                                title="Refresh"
                            >
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
                                // No rootPath needed based on your request
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
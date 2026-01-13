import { useState, useEffect, useRef } from "react";
import { FaFolder, FaFolderOpen, FaFileCode, FaTrash } from "react-icons/fa";
import { VscNewFile, VscNewFolder } from "react-icons/vsc";

interface FileNode {
    name: string;
    type: string; // "file" | "folder"
    children: FileNode[];
    path: string;
}

interface FolderProps {
    node: FileNode;
    selectedFolder: string | null;
    creatingConfig: { parentPath: string; type: "file" | "folder" } | null;
    onSelect: (node: FileNode) => void;
    onCreateSubmit: (name: string, parentPath: string, type: "file" | "folder") => void;
    onCancelCreate: () => void;
    onDelete: (path: string, type: "file" | "folder", name: string) => void;
}

export default function Folder({ 
    node, 
    selectedFolder, 
    creatingConfig,
    onSelect, 
    onCreateSubmit,
    onCancelCreate,
    onDelete 
}: FolderProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [newItemName, setNewItemName] = useState("");
    const inputRef = useRef<HTMLInputElement>(null);

    // Auto-expand if we are creating something inside this folder
    useEffect(() => {
        if (creatingConfig?.parentPath === node.path) {
            setIsOpen(true);
            // Focus happens automatically via autoFocus on input
        }
    }, [creatingConfig, node.path]);

    const handleClick = (e: React.MouseEvent) => {
        e.stopPropagation();
        onSelect(node);
        if (node.type === "folder") {
            setIsOpen(!isOpen);
        }
    };

    const handleDelete = (e: React.MouseEvent) => {
        e.stopPropagation();
        // Simple confirmation
        if (confirm(`Are you sure you want to delete ${node.name}?`)) {
            onDelete(node.path, node.type as "file"|"folder", node.name);
        }
    };

    const handleCreateKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === "Enter") {
            if (newItemName.trim()) {
                onCreateSubmit(newItemName, node.path, creatingConfig!.type);
                setNewItemName("");
            }
        } else if (e.key === "Escape") {
            onCancelCreate();
            setNewItemName("");
        }
    };

    const isCreatingHere = creatingConfig?.parentPath === node.path;
    const isSelected = selectedFolder === node.path;

    return (
        <div className="pl-3 select-none">
            {/* Row Item */}
            <div 
                className={`group flex items-center justify-between gap-2 py-1 px-2 cursor-pointer text-sm rounded transition-colors ${
                    isSelected ? "bg-slate-800 text-blue-400" : "hover:bg-slate-800/50 text-slate-400"
                }`}
                onClick={handleClick}
            >
                <div className="flex items-center gap-2 overflow-hidden">
                    <span className={isSelected ? "text-blue-400" : "text-slate-500"}>
                        {node.type === "folder" ? (
                            isOpen ? <FaFolderOpen /> : <FaFolder />
                        ) : (
                            <FaFileCode />
                        )}
                    </span>
                    <span className="truncate">{node.name}</span>
                </div>

                {/* Delete Button (Visible on Hover) */}
                <button 
                    onClick={handleDelete}
                    className="opacity-0 group-hover:opacity-100 hover:text-red-400 transition-opacity p-1"
                    title="Delete"
                >
                    <FaTrash size={10} />
                </button>
            </div>
            
            {/* Children & Creation Input */}
            {isOpen && (
                <div className="border-l border-slate-800 ml-2">
                    {/* Creation Input Field - VS Code Style */}
                    {isCreatingHere && (
                        <div className="pl-3 flex items-center gap-2 py-1">
                            <span className="text-slate-500 text-sm">
                                {creatingConfig.type === "folder" ? <FaFolder /> : <FaFileCode />}
                            </span>
                            <input
                                ref={inputRef}
                                autoFocus
                                type="text"
                                className="bg-slate-950 border border-blue-500 text-slate-300 text-sm px-1 py-0.5 w-full outline-none rounded-sm"
                                value={newItemName}
                                onChange={(e) => setNewItemName(e.target.value)}
                                onKeyDown={handleCreateKeyDown}
                                onBlur={() => {
                                    // Optional: Cancel on blur, or keep it open. VS Code usually confirms on enter.
                                    // onCancelCreate(); 
                                }}
                            />
                        </div>
                    )}

                    {/* Recursive Children */}
                    {node.children && node.children.map((child) => (
                        <Folder 
                            key={child.path} 
                            node={child} 
                            selectedFolder={selectedFolder}
                            creatingConfig={creatingConfig}
                            onSelect={onSelect}
                            onCreateSubmit={onCreateSubmit}
                            onCancelCreate={onCancelCreate}
                            onDelete={onDelete}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}
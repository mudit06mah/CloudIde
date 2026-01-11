import { useState } from "react";
import { FaFolder, FaFolderOpen, FaFileCode } from "react-icons/fa";

interface FileNode {
    name: string;
    type: string;
    children: FileNode[];
    path: string;
}

interface FolderProps {
    Node: FileNode;
    onSelectFile: (path: string) => void;
}

export default function Folder({ Node, onSelectFile }: FolderProps) {
    const [isOpen, setIsOpen] = useState(false);

    const handleClick = (e: React.MouseEvent) => {
        e.stopPropagation();
        if (Node.type === "folder") {
            setIsOpen(!isOpen);
        } else {
            onSelectFile(Node.path);
        }
    };

    return (
        <div className="pl-3 select-none">
            <div 
                className="flex items-center gap-2 py-1 hover:bg-slate-800 cursor-pointer text-sm rounded transition-colors"
                onClick={handleClick}
            >
                <span className="text-slate-500">
                    {Node.type === "folder" ? (
                        isOpen ? <FaFolderOpen /> : <FaFolder />
                    ) : (
                        <FaFileCode className="text-blue-400" />
                    )}
                </span>
                <span className={Node.type === "file" ? "text-slate-300" : "text-slate-100 font-medium"}>
                    {Node.name}
                </span>
            </div>
            
            {isOpen && Node.children && (
                <div className="border-l border-slate-800 ml-1.5">
                    {Node.children.map((child) => (
                        <Folder key={child.path} Node={child} onSelectFile={onSelectFile} />
                    ))}
                </div>
            )}
        </div>
    );
}
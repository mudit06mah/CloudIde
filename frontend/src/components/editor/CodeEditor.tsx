import { useRef } from 'react';
import Editor, { type OnChange } from '@monaco-editor/react';
import { useSocket } from '../../utils/Socket';

interface EditorProps {
    initialContent: string;
    filePath: string;
}

export default function CodeEditor({ initialContent, filePath }: EditorProps) {
    const { sendMessage } = useSocket();
    // Fix: Use ReturnType<typeof setTimeout> instead of NodeJS.Timeout
    const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    // Determine language based on extension
    const getLanguage = (path: string) => {
        if (path.endsWith('.js')) return 'javascript';
        if (path.endsWith('.ts') || path.endsWith('.tsx')) return 'typescript';
        if (path.endsWith('.py')) return 'python';
        if (path.endsWith('.go')) return 'go';
        if (path.endsWith('.cpp')) return 'cpp';
        if (path.endsWith('.html')) return 'html';
        if (path.endsWith('.css')) return 'css';
        if (path.endsWith('.json')) return 'json';
        return 'plaintext';
    };

    const handleEditorChange: OnChange = (value) => {
        if (value === undefined) return;

        // Debounce logic
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }

        timeoutRef.current = setTimeout(() => {
            // Encode content to base64 to avoid JSON breaking on special chars
            // Note: In a production app, consider using a library like 'js-base64' for better UTF-8 support
            const encodedContent = btoa(value);

            sendMessage("updateFile", {
                filePath: filePath, 
                content: encodedContent
            });
            
        }, 1000);
    };

    return (
        <Editor
            height="100%"
            theme="vs-dark"
            path={filePath}
            defaultLanguage={getLanguage(filePath)}
            defaultValue={initialContent}
            value={initialContent} // Controlled component
            onChange={handleEditorChange}
            options={{
                minimap: { enabled: false },
                fontSize: 14,
                fontFamily: "'JetBrains Mono', monospace",
                scrollBeyondLastLine: false,
                automaticLayout: true,
            }}
        />
    );
}
import { useEffect, useRef } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { AttachAddon } from 'xterm-addon-attach';

interface XtermProps {
    workspaceId: string;
}

const Xterm = ({ workspaceId }: XtermProps) => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const socketRef = useRef<WebSocket | null>(null);

    useEffect(() => {
        if (!workspaceId) return;

        // 1. Initialize Terminal
        const term = new Terminal({
            cursorBlink: true,
            theme: {
                background: "#020617",
                foreground: "#f8fafc",
            },
            fontSize: 14,
            fontFamily: "'JetBrains Mono', monospace",
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        
        if (terminalRef.current) {
            term.open(terminalRef.current);
            fitAddon.fit();
        }

        // 2. Connect to WebSocket
        // FIX: Added 'workspaceId' query param so backend can initialize the K8s Client
        const socket = new WebSocket(
            `ws://localhost:8080/ws?type=terminal&pod=shell-${workspaceId}&workspaceId=${workspaceId}`
        );
        
        socket.onopen = () => {
            // Attach the socket to xterm
            const attachAddon = new AttachAddon(socket);
            term.loadAddon(attachAddon);
            
            // Send a resize immediately to fix layout
            const dims = fitAddon.proposeDimensions();
            if (dims) {
                fitAddon.fit();
                // Optional: Send initial resize opcode if your backend expects it
                socket.send(JSON.stringify({ 
                    op: "resize", 
                    cols: dims.cols, 
                    rows: dims.rows 
                }));
            }
        };

        socket.onerror = (err) => {
            console.error("Terminal WebSocket Error:", err);
            term.write("\r\n\x1b[31mConnection Error. Please refresh.\x1b[0m\r\n");
        };

        socketRef.current = socket;

        // 3. Handle Resize
        const handleResize = () => {
            if (socket.readyState === WebSocket.OPEN) {
                const dims = fitAddon.proposeDimensions();
                if (dims) {
                    fitAddon.fit();
                    socket.send(JSON.stringify({ 
                        op: "resize", 
                        cols: dims.cols, 
                        rows: dims.rows 
                    }));
                }
            }
        };
        window.addEventListener('resize', handleResize);

        // 4. Cleanup
        return () => {
            window.removeEventListener('resize', handleResize);
            socket.close();
            term.dispose();
        };
    }, [workspaceId]);

    return <div ref={terminalRef} className="h-full w-full bg-slate-950 px-2" />;
};

export default Xterm;
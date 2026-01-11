import { useEffect, useRef } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";

export default function Xterm({ workspaceId }: { workspaceId: string }) {
    const terminalRef = useRef<HTMLDivElement | null>(null);
    const wsRef = useRef<WebSocket | null>(null);

    useEffect(() => {
        // Connect to the specific terminal endpoint
        // Assuming pod name is 'shell-<workspaceId>' based on your backend logic
        const ws = new WebSocket(`ws://localhost:8080/ws?type=terminal&pod=shell-${workspaceId}`);
        wsRef.current = ws;

        const term = new Terminal({
            cursorBlink: true,
            theme: {
                background: '#000000',
                foreground: '#ffffff',
            },
            fontFamily: "'JetBrains Mono', monospace",
        });
        
        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);

        if (terminalRef.current) {
            term.open(terminalRef.current);
            fitAddon.fit();
        }

        // Send input to backend
        term.onData((data) => {
            if (ws.readyState === WebSocket.OPEN) {
                // Protocol: Op "stdin", Data string
                const msg = JSON.stringify({ op: "stdin", data: data });
                ws.send(msg);
            }
        });

        // Handle resize
        term.onResize((size) => {
            if (ws.readyState === WebSocket.OPEN) {
                // Protocol: Op "resize", Cols, Rows
                const msg = JSON.stringify({ op: "resize", cols: size.cols, rows: size.rows });
                ws.send(msg);
            }
        });

        ws.onmessage = (event) => {
            // Backend sends raw text (not JSON) for stdout based on wsWriter?
            // Wait, terminal.go uses `ws.WriteMessage(websocket.TextMessage, p)`.
            // It sends raw bytes from the PTY.
            if(typeof event.data === 'string') {
                 term.write(event.data);
            } else {
                 // Handle blob if necessary
                 const reader = new FileReader();
                 reader.onload = () => {
                     term.write(reader.result as string);
                 };
                 reader.readAsText(event.data);
            }
        };

        const handleResizeWindow = () => fitAddon.fit();
        window.addEventListener("resize", handleResizeWindow);

        return () => {
            window.removeEventListener("resize", handleResizeWindow);
            term.dispose();
            ws.close();
        };

    }, [workspaceId]);

    return (
        <div ref={terminalRef} className="w-full h-full bg-black pl-2 pt-2" />
    );
}
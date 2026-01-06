import { useEffect, useRef } from "react";
import { Terminal } from "xterm";

export default function Xterm({workspaceId}:{workspaceId: string}){
    const terminalRef = useRef<HTMLDivElement | null>(null)
    useEffect(() => {
        //open websocket:
        const termSocket = new WebSocket("ws://localhost:8080/ws?type=terminal&pod=shell-" + workspaceId);
        
        //open terminal:
        const term = new Terminal()
        if(terminalRef.current){
            term.open(terminalRef.current)
        }

        term.onData((data) => {
            termSocket.send(data)
        })
        term.onResize((data) => {
            const msg = JSON.stringify(data)
            termSocket.send(msg)
        })

        termSocket.onmessage = (msg:MessageEvent) => {
            term.write(msg.data)
        }

        //cleanup
        return () => {
            termSocket.close();
        }

    },[workspaceId]);

    return(
        <div ref={terminalRef} className="w-full h-full bg-black text-white">
        </div>
    )
}
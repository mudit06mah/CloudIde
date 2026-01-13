import { createContext, useContext, useEffect, useState, useRef, type PropsWithChildren } from "react";

interface SocketContextType {
    socket: WebSocket | null;
    sendMessage: (type: string, payload: any) => void;
    // Subscribe to messages. Returns an unsubscribe function.
    subscribe: (event: string, callback: (payload: any) => void) => () => void;
}

const WsContext = createContext<SocketContextType | null>(null);

export const useSocket = () => {
    const context = useContext(WsContext);
    if (!context) {
        throw new Error("useSocket must be used within a SocketProvider");
    }
    return context;
};

export function SocketProvider({ children }: PropsWithChildren<{}>) {
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const listeners = useRef<Map<string, Set<(payload: any) => void>>>(new Map());

    useEffect(() => {
        const ws = new WebSocket("ws://localhost:8080/ws"); // Ensure port matches your backend

        ws.onopen = () => {
            console.log("Connected to WS Server");
            setSocket(ws);
        };

        ws.onmessage = (event) => {
            try {
                const response = JSON.parse(event.data);
                // Backend sends: { success: bool, message: string, payload: any }
                // We use the 'message' field as the event type trigger
                const eventType = response.message; 
                
                if (listeners.current.has(eventType)) {
                    listeners.current.get(eventType)?.forEach((cb) => cb(response.payload));
                }
            } catch (e) {
                console.error("Failed to parse WS message", e);
            }
        };

        ws.onclose = () => {
            console.log("WebSocket connection closed");
            setSocket(null);
        };

        return () => {
            ws.close();
        };
    }, []);

    const sendMessage = (type: string, payload: any) => {
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type, payload }));
        } else {
            console.warn("Socket not connected");
        }
    };

    const subscribe = (event: string, callback: (payload: any) => void) => {
        if (!listeners.current.has(event)) {
            listeners.current.set(event, new Set());
        }
        listeners.current.get(event)?.add(callback);

        return () => {
            listeners.current.get(event)?.delete(callback);
        };
    };

    return (
        <WsContext.Provider value={{ socket, sendMessage, subscribe }}>
            {children}
        </WsContext.Provider>
    );
}
``
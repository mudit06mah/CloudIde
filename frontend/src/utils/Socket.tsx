import {  createContext, useEffect, type PropsWithChildren } from "react";
import { useState } from "react";

interface wscontext{
    socket: WebSocket|null,
    messages: string[]
}

const WsContext =  createContext<wscontext>({socket:null, messages:[]})

export function Socket(props: PropsWithChildren<{}>){

    const [socket, setSocket] = useState<WebSocket | null>(null);
    const [messages, setMessages] = useState<string[]>([]); 

    useEffect(() => {
        //todo: replace with env variable
        const ws = new WebSocket("ws://localhost:8000/ws");
        setSocket(ws);
        
        ws.onmessage = (event) => {
            setMessages((prevMessages) => [...prevMessages, event.data]);
        };
        
        ws.onclose = () => {
            console.log("WebSocket connection closed");
        };

        //cleanup
        return () => { ws.close(); }        

    }, []);

    return(
        <WsContext.Provider value={{socket,messages}}>
            {props.children}
        </WsContext.Provider>
    );
}

export {WsContext}

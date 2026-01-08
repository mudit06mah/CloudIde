interface FileNode{
    name: string,
    type: string,
    children: FileNode[],
    path: string
}

export default function Folder({Node}:{Node: FileNode}){
    const children = Node.children.map((child) =>{
        if(child.type == "file"){
            return <li key={child.path}>{child.name}</li>
        }
        else{
            return <li key={child.path}>
                <Folder Node={child}></Folder>
            </li>
        }
    })

    return(
        <div>
            <p>{Node.name}</p>
            <ul>
                {children}
            </ul>
        </div>
    )
}
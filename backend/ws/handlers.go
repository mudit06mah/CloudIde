package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"github.com/mudit06mah/CloudIde/aws"
	"github.com/mudit06mah/CloudIde/k8s"
)

type Response struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WSWriter struct {
	Conn *websocket.Conn
}

type FileNode struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Children []FileNode `json:"children"`
	Path     string     `json:"path"`
}

func (w *WSWriter) Write(p []byte) (n int, err error) {
	response := Response{
		Success: true,
		Message: "terminal:output",
		Payload: json.RawMessage(fmt.Sprintf("%q", string(p))),
	}

	msg, err := json.Marshal(response)
	if err != nil {
		return 0, err
	}

	err = w.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

const namespace string = "cloud-ide"

var validate = validator.New()
var conn *websocket.Conn
var workspaceId string
var client *k8s.Client

// helper functions:
func createWorkspaceId(size int) string {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"
	id := ""

	for i := 0; i < size; i++ {
		id += string(charset[rand.Intn(len(charset))])
	}

	if _, exists := workspaces[id]; exists {
		return createWorkspaceId(size)
	}

	return id
}

func sendResponse(success bool, message string, payload json.RawMessage) {
	response := Response{
		Success: success,
		Message: message,
		Payload: payload,
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshalling response:", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, respBytes)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}

}

func messageHandler(con *websocket.Conn, rawMsg []byte) {
	var msg Message
	err := json.Unmarshal(rawMsg, &msg)
	if err != nil {
		fmt.Println("Error unmarshalling message:", err)
		sendResponse(false, "Error unmarshalling message: "+err.Error(), nil)
		return
	}

	conn = con

	switch msg.Type {

	case "initProject":
		handleCreateProject(msg.Payload)

	case "createFile":
		handleCreateFile(msg.Payload)

	case "getFile":
		handleGetFile(msg.Payload)

	case "deleteFile":
		handleDeleteFile(msg.Payload)

	case "createFolder":
		handleCreateFolder(msg.Payload)

	case "deleteFolder":
		handleDeleteFolder(msg.Payload)

	case "updateFile":
		handleUpdateFile(msg.Payload)

	case "requestTerminal":
		handleRequestTerminal(msg.Payload)

	case "getTree":
		handleGetTree(msg.Payload)

	default:
		fmt.Println("Unknown message type:", msg.Type)
		sendResponse(false, "Unknown message type: "+msg.Type, nil)

	}
}

func handleCreateProject(payload json.RawMessage) {
	type CreateProjectData struct {
		ProjectType string `json:"projectType" validate:"required,oneof=python nodejs golang cpp react"`
	}
	var data CreateProjectData

	//Unmarshal
	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling createProject payload:", err)
		sendResponse(false, "Error unmarshalling createProject payload: "+err.Error(), nil)
		return
	}

	//Validate
	if err = validate.Struct(data); err != nil {
		fmt.Println("Validation error:", err)
		sendResponse(false, "Validation error: "+err.Error(), nil)
		return
	}

	workspaceId = createWorkspaceId(10)
	workspaces[workspaceId] = data.ProjectType

	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
	//os.Chown(cacheDir, 1500, 1500)

	_, err = aws.DownloadTemplate(data.ProjectType, workspaceId)
	if err != nil {
		fmt.Println("Error downloading template:", err)
		sendResponse(false, "Error downloading template: "+err.Error(), nil)
		return
	}

	//create client:
	ctx := context.Background()
	client, err = k8s.NewK8sClient(workspaceId)
	if err != nil {
		fmt.Println("Error creating k8s client:", err)
		sendResponse(false, "Error creating k8s client: "+err.Error(), nil)
		return
	}

	//create shell pod:
	var manifests [][]byte
	manifests, err = client.RenderProjectResources(data.ProjectType)
	if err != nil {
		fmt.Println("Error obtaining manifest files:", err)
		sendResponse(false, "Error obtaining manifest files:"+err.Error(), nil)
	}

	for _, manifest := range manifests {
		err = client.ApplyManifest(ctx, manifest)
		if err != nil {
			fmt.Println("Error applying manifest:", err)
			sendResponse(false, "Error applying manifest:"+err.Error(), nil)
		}
	}

	_, err = client.WaitForPodByLabel(
		ctx,
		namespace,
		fmt.Sprintf("workspace=%s", workspaceId),
		300*time.Second,
	)
	if err != nil {
		fmt.Println("Error obtaining manifest files:", err)
		sendResponse(false, "Error obtaining manifest files:"+err.Error(), nil)
	}

	type ProjectPayload struct {
		WorkspaceId string `json:"workspaceId"`
		Tree FileNode `json:"fileNode"`
	}

	tree,_ := generateTree(cacheDir,workspaceId)

	response, _ := json.Marshal(ProjectPayload{WorkspaceId: workspaceId, Tree: tree})
	sendResponse(true, "Project created successfully", response)
}

func handleCreateFile(payload json.RawMessage) {
	type CreateFile struct {
		FileName string `json:"fileName" validate:"required"`
		FilePath string `json:"filePath" validate:"required"`
	}
	var data CreateFile
	//Unmarshal
	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling createFile payload:", err)
		sendResponse(false, "Error unmarshalling createFile payload: "+err.Error(), nil)
		return
	}
	//Validate
	if err = validate.Struct(data); err != nil {
		fmt.Println("Validation error:", err)
		sendResponse(false, "Validation error: "+err.Error(), nil)
		return
	}

	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
	localPath := filepath.Join(cacheDir, data.FilePath, data.FileName)

	file, err := os.Create(localPath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		sendResponse(false, "Error creating file: "+err.Error(), nil)
		return
	}

	sendResponse(true, "File created successfully", nil)
	defer file.Close()
}

func handleGetFile(payload json.RawMessage) {
	type GetFile struct {
		FileName string `json:"fileName" validate:"required"`
		FilePath string `json:"filePath" validate:"required"`
	}
	var data GetFile
	//Unmarshal
	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling getFile payload:", err)
		sendResponse(false, "Error unmarshalling getFile payload: "+err.Error(), nil)
		return
	}
	//Validate
	if err = validate.Struct(data); err != nil {
		fmt.Println("Validation error:", err)
		sendResponse(false, "Validation error: "+err.Error(), nil)
		return
	}

	//check if file exists
	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
	localPath := filepath.Join(cacheDir, data.FilePath, data.FileName)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		fmt.Println("File does not exist:", localPath)
		sendResponse(false, "File does not exist: "+localPath, nil)
		return
	}

	fileContent, err := os.ReadFile(localPath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		sendResponse(false, "Error reading file: "+err.Error(), nil)
		return
	}

	//marshaling file content
	type FileContent struct {
		Content string `json:"content"`
	}

	payloadData, err := json.Marshal(FileContent{Content: string(fileContent)})
	if err != nil {
		fmt.Println("Error marshalling file content:", err)
		sendResponse(false, "Error marshalling file content: "+err.Error(), nil)
		return
	}

	sendResponse(true, "File retrieved successfully", payloadData)
}

func handleUpdateFile(payload json.RawMessage) {
    // UPDATED: Now accepts full content instead of line-by-line
    type UpdateFile struct {
        FileName string `json:"fileName" validate:"required"`
        FilePath string `json:"filePath" validate:"required"`
        Content  string `json:"content" validate:"required"` // Base64 encoded full content
    }

    var data UpdateFile
    // Unmarshal
    err := json.Unmarshal(payload, &data)
    if err != nil {
        fmt.Println("Error unmarshalling updateFile payload:", err)
        sendResponse(false, "Error unmarshalling updateFile payload: "+err.Error(), nil)
        return
    }
    // Validate
    if err = validate.Struct(data); err != nil {
        fmt.Println("Validation error:", err)
        sendResponse(false, "Validation error: "+err.Error(), nil)
        return
    }

    // Construct local path
    cacheDir := filepath.Join(os.Getenv("CACHE_DIR"), workspaceId)
    localPath := filepath.Join(cacheDir, data.FilePath, data.FileName)

    // Decode Base64 Content
    decodedContent, err := base64.StdEncoding.DecodeString(data.Content)
    if err != nil {
        fmt.Println("Error decoding file content:", err)
        sendResponse(false, "Error decoding file content: "+err.Error(), nil)
        return
    }

    // Write file (Overwrite)
    // 0644 provides read/write for owner, read-only for others
    err = os.WriteFile(localPath, decodedContent, 0644)
    if err != nil {
        fmt.Println("Error writing file:", err)
        sendResponse(false, "Error writing file: "+err.Error(), nil)
        return
    }

    sendResponse(true, "File updated successfully", nil)
}

func handleDeleteFile(payload json.RawMessage) {
	var data struct {
		FileName string `json:"fileName" validate:"required"`
		FilePath string `json:"filePath" validate:"required"`
	}
	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling deleteFile payload:", err)
		sendResponse(false, "Error unmarshalling deleteFile payload: "+err.Error(), nil)
		return
	}

	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
	localPath := filepath.Join(cacheDir, data.FilePath, data.FileName)
	err = os.Remove(localPath)
	if err != nil {
		fmt.Println("Error deleting file:", err)
		sendResponse(false, "Error deleting file: "+err.Error(), nil)
		return
	}

	sendResponse(true, "File deleted successfully", nil)
}

func handleCreateFolder(payload json.RawMessage) {
	var data struct {
		FolderName string `json:"folderName" validate:"required"`
		FolderPath string `json:"folderPath" validate:"required"`
	}

	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling createFolder payload:", err)
		sendResponse(false, "Error unmarshalling createFolder payload: "+err.Error(), nil)
		return
	}

	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
	localPath := filepath.Join(cacheDir, data.FolderPath, data.FolderName)
	err = os.MkdirAll(localPath, 0755)
	if err != nil {
		fmt.Println("Error creating folder:", err)
		sendResponse(false, "Error creating folder: "+err.Error(), nil)
		return
	}

	sendResponse(true, "Folder created successfully", nil)

}

func handleDeleteFolder(payload json.RawMessage) {
	var data struct {
		FolderPath string `json:"filePath" validate:"required"`
	}
	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling deleteFile payload:", err)
		sendResponse(false, "Error unmarshalling deleteFolder payload: "+err.Error(), nil)
		return
	}

	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
	localPath := filepath.Join(cacheDir, data.FolderPath)
	err = os.RemoveAll(localPath)
	if err != nil {
		fmt.Println("Error deleting file:", err)
		sendResponse(false, "Error deleting folder: "+err.Error(), nil)
		return
	}

	sendResponse(true, "Folder deleted successfully", nil)
}

func handleRequestTerminal(payload json.RawMessage) {
	var data struct {
		Instruction string `json:"instruction" validate:"required"`
	}

	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("Error unmarshalling requestTerminal payload:", err)
		sendResponse(false, "Error unmarshalling requestTerminal payload: "+err.Error(), nil)
		return
	}

	if client == nil {
		fmt.Println("K8s client not initialized.")
		sendResponse(false, "K8s client not initialized", nil)
		return
	}

	ctx := context.Background()
	podName, err := client.WaitForPodByLabel(ctx, namespace, fmt.Sprintf("workspace=%s", workspaceId), 1*time.Second)
	if err != nil {
		fmt.Println("Error finding pod:", err)
		sendResponse(false, "Error finding pod: "+err.Error(), nil)
		return
	}

	wsWriter := &WSWriter{Conn: conn}

	cmd := []string{"bin/bash", "-c", data.Instruction}

	err = client.ExecToPod(ctx, namespace, podName, "shell", cmd, nil, wsWriter, wsWriter, false)
	if err != nil {
		fmt.Println("Error executing command:", err)
		sendResponse(false, "Error executing command:"+err.Error(), nil)
		return
	}
}

func handleGetTree(payload json.RawMessage) {
	var data struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	
	err := json.Unmarshal(payload, &data)
	if err != nil{
		fmt.Println("Error Unmarshaling handleGetTree payload: ",err)
		sendResponse(false,"Error Unmarshaling handleGetTree payload: "+err.Error(), nil)
		return
	}

	Tree, err := generateTree(data.Path, data.Name)
	if err != nil{
		fmt.Println("Error Walking Path/Path Does not exist: ", err)
		sendResponse(false, "Error Walking Path/Path Does not exist: "+err.Error(), nil)
		return
	}

	response, err := json.Marshal(Tree)
	if err != nil{
		fmt.Println("Error Marshaling Tree: ",Tree,"\nError:", err)
		sendResponse(false,"Error Marshaling Tree: "+err.Error(),nil)
		return	
	}

	sendResponse(true,"Succesfully generated tree",response)
}

func generateTree(path string, name string) (FileNode, error) {
	enteries, err := os.ReadDir(path)

	if err != nil {
		fmt.Println("Path does not exist: ", path)
		return FileNode{}, err
	}

	var Tree FileNode
	Tree.Name = name	
	Tree.Type = "folder"
	Tree.Path = path

	for _, entry := range enteries {
		var child FileNode
		if entry.IsDir() {
			child,_ = generateTree(filepath.Join(path,entry.Name()),entry.Name())
		} else {
			child.Name = entry.Name()
			child.Type = "file"
			child.Children = nil
			child.Path = filepath.Join(path,entry.Name())
		}
		Tree.Children = append(Tree.Children, child)
	}

	return Tree, nil
}

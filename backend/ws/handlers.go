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

// --- Structs ---

type Response struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type FileNode struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Children []FileNode `json:"children"`
	Path     string     `json:"path"`
}

// WSWriter adapter for K8s exec
type WSWriter struct {
	Conn *websocket.Conn
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

// --- Session Logic (NO GLOBALS) ---

var validate = validator.New()

const namespace = "cloud-ide"

// Session holds the state for ONE specific user connection
type Session struct {
	ProjectType string
	Conn        *websocket.Conn
	WorkspaceID string
	K8sClient   *k8s.Client
}

func NewSession(conn *websocket.Conn) *Session {
	return &Session{
		Conn: conn,
	}
}

// HandleMessage routes incoming messages for THIS session
func (s *Session) HandleMessage(rawMsg []byte) {
	var msg Message
	err := json.Unmarshal(rawMsg, &msg)
	if err != nil {
		fmt.Println("Error unmarshalling message:", err)
		s.sendResponse(false, "Error unmarshalling message: "+err.Error(), nil)
		return
	}

	switch msg.Type {
	case "initProject":
		s.handleCreateProject(msg.Payload)
	case "createFile":
		s.handleCreateFile(msg.Payload)
	case "getFile":
		s.handleGetFile(msg.Payload)
	case "deleteFile":
		s.handleDeleteFile(msg.Payload)
	case "createFolder":
		s.handleCreateFolder(msg.Payload)
	case "deleteFolder":
		s.handleDeleteFolder(msg.Payload)
	case "updateFile":
		s.handleUpdateFile(msg.Payload)
	case "requestTerminal":
		s.handleRequestTerminal(msg.Payload)
	case "getTree":
		s.handleGetTree(msg.Payload)
	case "stopWorkspace":
		s.handleStopWorkspace(msg.Payload)
	default:
		fmt.Println("Unknown message type:", msg.Type)
		s.sendResponse(false, "Unknown message type: "+msg.Type, nil)
	}
}

// helper functions:
func (s *Session) sendResponse(success bool, message string, payload json.RawMessage) {
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
	err = s.Conn.WriteMessage(websocket.TextMessage, respBytes)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func createWorkspaceId(size int) string {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"
	id := ""
	for i := 0; i < size; i++ {
		id += string(charset[rand.Intn(len(charset))])
	}
	return id
}

func (s *Session) handleCreateProject(payload json.RawMessage) {
	type CreateProjectData struct {
		ProjectType string `json:"projectType" validate:"required,oneof=python nodejs golang cpp react"`
	}
	var data CreateProjectData
	if err := json.Unmarshal(payload, &data); err != nil {
		s.sendResponse(false, "Error unmarshalling", nil)
		return
	}
	if err := validate.Struct(data); err != nil {
		s.sendResponse(false, "Validation error: "+err.Error(), nil)
		return
	}

	s.WorkspaceID = createWorkspaceId(10)
	s.ProjectType = data.ProjectType
	currentCachePath := filepath.Join(os.Getenv("CACHE_DIR"), s.WorkspaceID)

	if err := os.MkdirAll(currentCachePath, 0755); err != nil {
		s.sendResponse(false, "Error creating cache dir: "+err.Error(), nil)
		return
	}

	if _, err := aws.DownloadTemplate(data.ProjectType, s.WorkspaceID); err != nil {
		s.sendResponse(false, "Error downloading template: "+err.Error(), nil)
		return
	}

	var err error
	s.K8sClient, err = k8s.NewK8sClient(s.WorkspaceID)
	if err != nil {
		s.sendResponse(false, "Error creating k8s client: "+err.Error(), nil)
		return
	}

	ctx := context.Background()
	manifests, err := s.K8sClient.RenderProjectResources(data.ProjectType)
	if err != nil {
		s.sendResponse(false, "Error obtaining manifests: "+err.Error(), nil)
		return
	}
	for _, manifest := range manifests {
		s.K8sClient.ApplyManifest(ctx, manifest)
	}

	_, err = s.K8sClient.WaitForPodByLabel(ctx, namespace, fmt.Sprintf("workspace=%s", s.WorkspaceID), 300*time.Second)
	if err != nil {
		s.sendResponse(false, "Error waiting for pod: "+err.Error(), nil)
		return
	}

	tree, _ := generateTree(currentCachePath, s.WorkspaceID)
	type ProjectPayload struct {
		WorkspaceId string   `json:"workspaceId"`
		Tree        FileNode `json:"fileNode"`
	}
	response, _ := json.Marshal(ProjectPayload{WorkspaceId: s.WorkspaceID, Tree: tree})
	s.sendResponse(true, "Project created successfully", response)
}

func (s *Session) handleCreateFile(payload json.RawMessage) {
	type CreateFile struct {
		FileName string `json:"fileName"`
		FilePath string `json:"filePath"`
	}
	var data CreateFile
	if err := json.Unmarshal(payload, &data); err != nil {
		fmt.Println("Error unmarshalling create file payload:", err)
		s.sendResponse(false, "Error unmarshalling payload: "+err.Error(), nil)
		return
	}

	localPath := filepath.Join(data.FilePath, data.FileName)
	file, err := os.Create(localPath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		s.sendResponse(false, "Error creating file: "+err.Error(), nil)
		return
	}
	if err := file.Close(); err != nil {
		fmt.Println("Error closing file:", err)
		s.sendResponse(false, "Error closing file: "+err.Error(), nil)
		return
	}
	s.sendResponse(true, "File created successfully", nil)
}

func (s *Session) handleGetFile(payload json.RawMessage) {
	type GetFile struct {
		FilePath string `json:"filePath"`
	}
	var data GetFile
	if err := json.Unmarshal(payload, &data); err != nil {
		fmt.Println("Error unmarshalling get file payload:", err)
		s.sendResponse(false, "Error unmarshalling payload: "+err.Error(), nil)
		return
	}

	if _, err := os.Stat(data.FilePath); os.IsNotExist(err) {
		fmt.Println("File does not exist:", data.FilePath)
		s.sendResponse(false, "File does not exist", nil)
		return
	}
	content, err := os.ReadFile(data.FilePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		s.sendResponse(false, "Error reading file: "+err.Error(), nil)
		return
	}
	type FileContent struct {
		Content string `json:"content"`
	}
	resp, err := json.Marshal(FileContent{Content: string(content)})
	if err != nil {
		fmt.Println("Error marshalling file content:", err)
		s.sendResponse(false, "Error marshalling file content: "+err.Error(), nil)
		return
	}
	s.sendResponse(true, "File retrieved successfully", resp)
}

func (s *Session) handleUpdateFile(payload json.RawMessage) {
	type UpdateFile struct {
		FilePath string `json:"filePath"`
		Content  string `json:"content"`
	}
	var data UpdateFile
	if err := json.Unmarshal(payload, &data); err != nil {
		fmt.Println("Error unmarshalling update file payload:", err)
		s.sendResponse(false, "Error unmarshalling payload: "+err.Error(), nil)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(data.Content)
	if err != nil {
		fmt.Println("Error decoding content:", err)
		s.sendResponse(false, "Error decoding content: "+err.Error(), nil)
		return
	}

	if err := os.WriteFile(data.FilePath, decoded, 0644); err != nil {
		fmt.Println("Error writing file:", err)
		s.sendResponse(false, "Error writing file: "+err.Error(), nil)
		return
	}
	s.sendResponse(true, "File updated successfully", nil)
}

func (s *Session) handleDeleteFile(payload json.RawMessage) {
	var data struct {
		FileName string `json:"fileName"`
		FilePath string `json:"filePath"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		fmt.Println("Error unmarshalling delete file payload:", err)
		s.sendResponse(false, "Error unmarshalling payload: "+err.Error(), nil)
		return
	}
	filePath := filepath.Join(data.FilePath, data.FileName)
	if err := os.Remove(filePath); err != nil {
		fmt.Println("Error deleting file:", err)
		s.sendResponse(false, "Error deleting file: "+err.Error(), nil)
		return
	}
	s.sendResponse(true, "File deleted successfully", nil)
}

func (s *Session) handleCreateFolder(payload json.RawMessage) {
	var data struct {
		FolderName string `json:"folderName"`
		FolderPath string `json:"folderPath"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		fmt.Println("Error unmarshalling create folder payload:", err)
		s.sendResponse(false, "Error unmarshalling payload: "+err.Error(), nil)
		return
	}
	folderPath := filepath.Join(data.FolderPath, data.FolderName)
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		fmt.Println("Error creating folder:", err)
		s.sendResponse(false, "Error creating folder: "+err.Error(), nil)
		return
	}
	s.sendResponse(true, "Folder created successfully", nil)
}

func (s *Session) handleDeleteFolder(payload json.RawMessage) {
	var data struct {
		FolderPath string `json:"folderPath"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		fmt.Println("Error unmarshalling delete folder payload:", err)
		s.sendResponse(false, "Error unmarshalling payload: "+err.Error(), nil)
		return
	}
	if err := os.RemoveAll(data.FolderPath); err != nil {
		fmt.Println("Error deleting folder:", err)
		s.sendResponse(false, "Error deleting folder: "+err.Error(), nil)
		return
	}
	s.sendResponse(true, "Folder deleted successfully", nil)
}

func (s *Session) handleRequestTerminal(payload json.RawMessage) {
	var data struct {
		Instruction string `json:"instruction"`
	}
	json.Unmarshal(payload, &data)

	if s.K8sClient == nil {
		s.sendResponse(false, "K8s client not initialized", nil)
		return
	}

	ctx := context.Background()
	podName, err := s.K8sClient.WaitForPodByLabel(ctx, namespace, fmt.Sprintf("workspace=%s", s.WorkspaceID), 1*time.Second)
	if err != nil {
		s.sendResponse(false, "Error finding pod", nil)
		return
	}

	wsWriter := &WSWriter{Conn: s.Conn}
	cmd := []string{"bin/bash", "-c", data.Instruction}
	s.K8sClient.ExecToPod(ctx, namespace, podName, "shell", cmd, nil, wsWriter, wsWriter, false)
}

func (s *Session) handleGetTree(payload json.RawMessage) {
	var data struct {
		WorkspaceId string `json:"workspaceId"`
	}
	json.Unmarshal(payload, &data)

	targetId := data.WorkspaceId
	if targetId == "" {
		targetId = s.WorkspaceID
	}
	if targetId == "" {
		s.sendResponse(false, "WorkspaceId not found", nil)
		return
	}

	if s.WorkspaceID == "" {
		s.WorkspaceID = targetId
	}

	path := filepath.Join(os.Getenv("CACHE_DIR"), targetId)
	tree, err := generateTree(path, targetId)
	if err != nil {
		s.sendResponse(false, "Error generating tree", nil)
		return
	}

	type getTreeResponse struct {
		Tree FileNode `json:"tree"`
	}
	resp, _ := json.Marshal(getTreeResponse{Tree: tree})
	s.sendResponse(true, "Succesfully generated tree", resp)
}

func generateTree(path string, name string) (FileNode, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return FileNode{}, err
	}
	var Tree FileNode
	Tree.Name = name
	Tree.Type = "folder"
	Tree.Path = path

	for _, entry := range entries {
		//ignore node_modules and .git:
		if entry.Name() == "node_modules" || entry.Name() == ".git" {
			continue
		}

		var child FileNode
		if entry.IsDir() {
			child, _ = generateTree(filepath.Join(path, entry.Name()), entry.Name())
		} else {
			child.Name = entry.Name()
			child.Type = "file"
			child.Path = filepath.Join(path, entry.Name())
		}
		Tree.Children = append(Tree.Children, child)
	}
	return Tree, nil
}

func (s *Session) handleStopWorkspace(payload json.RawMessage) {
	var data struct {
		WorkspaceId string `json:"workspaceId"`
	}
	json.Unmarshal(payload, &data)

	// Determine ID
	targetId := s.WorkspaceID
	if targetId == "" {
		targetId = data.WorkspaceId
	}

	if targetId == "" {
		s.sendResponse(false, "Workspace ID missing", nil)
		return
	}

	//cleanup function:
	err := s.cleanup(targetId)
	if err != nil{
		fmt.Println("Error Cleaning Up: ",err)
		s.sendResponse(false, "Error Cleaning Up:"+err.Error(),nil)
	}

	fmt.Printf("Workspace %s stopped and cleaned up.\n", targetId)
	s.sendResponse(true, "Workspace stopped successfully", nil)
}

func (s *Session)cleanup(targetId string) error{
	ctx := context.Background()
	resourceName := fmt.Sprintf("shell-%s", targetId)

	//delete resources:
	if s.ProjectType == "react" || s.ProjectType == "nodejs" {
		if err := s.K8sClient.DeleteResource(ctx, "Ingress", resourceName, namespace); err != nil {
			fmt.Println("Error deleting ingress", err)
			return err
		}

		if err := s.K8sClient.DeleteResource(ctx, "Service", resourceName, namespace); err != nil {
			fmt.Println("Error deleting ingress", err)
			return err
		}
	}

	if err := s.K8sClient.DeleteResource(ctx, "Pod", resourceName, namespace); err != nil {
		fmt.Println("Error deleting pod:", err)
		return err

	}

	//delete Local Cache
	cacheDir := filepath.Join(os.Getenv("CACHE_DIR"), targetId)
	if err := os.RemoveAll(cacheDir); err != nil {
		fmt.Println("Error deleting cache:", err)
		return err
	}

	return nil
}

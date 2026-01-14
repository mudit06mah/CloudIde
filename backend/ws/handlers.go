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

// --- Session Logic ---

var validate = validator.New()

// Session holds the state for ONE specific user connection
type Session struct {
	Conn        *websocket.Conn
	WorkspaceID string
	K8sClient   *k8s.Client
}

// NewSession creates a new session wrapper for a connection
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
	default:
		fmt.Println("Unknown message type:", msg.Type)
		s.sendResponse(false, "Unknown message type: "+msg.Type, nil)
	}
}

// --- Helper Methods ---

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
	// Write to THIS session's connection
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
	// Note: We removed the global map check for simplicity/concurrency safety.
	// In production, check DB or Redis here.
	return id
}

// --- Session Handlers ---

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

	// 1. Set Session State
	s.WorkspaceID = createWorkspaceId(10)
	currentCachePath := filepath.Join(os.Getenv("CACHE_DIR"), s.WorkspaceID)

	// 2. Setup FS
	if err := os.MkdirAll(currentCachePath, 0755); err != nil {
		s.sendResponse(false, "Error creating cache dir: "+err.Error(), nil)
		return
	}

	// 3. Download Template
	if _, err := aws.DownloadTemplate(data.ProjectType, s.WorkspaceID); err != nil {
		s.sendResponse(false, "Error downloading template: "+err.Error(), nil)
		return
	}

	// 4. Setup K8s Client for THIS Session
	var err error
	s.K8sClient, err = k8s.NewK8sClient(s.WorkspaceID)
	if err != nil {
		s.sendResponse(false, "Error creating k8s client: "+err.Error(), nil)
		return
	}

	// 5. Deploy Resources
	ctx := context.Background()
	manifests, err := s.K8sClient.RenderProjectResources(data.ProjectType)
	if err != nil {
		s.sendResponse(false, "Error obtaining manifests: "+err.Error(), nil)
		return
	}
	for _, manifest := range manifests {
		s.K8sClient.ApplyManifest(ctx, manifest)
	}

	// 6. Wait for Pod
	namespace := "cloud-ide" // hardcoded for now, or match your server.go
	_, err = s.K8sClient.WaitForPodByLabel(ctx, namespace, fmt.Sprintf("workspace=%s", s.WorkspaceID), 300*time.Second)
	if err != nil {
		s.sendResponse(false, "Error waiting for pod: "+err.Error(), nil)
		return
	}

	// 7. Return Tree
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
	json.Unmarshal(payload, &data)

	localPath := filepath.Join(data.FilePath, data.FileName)
	file, err := os.Create(localPath)
	if err != nil {
		s.sendResponse(false, "Error creating file: "+err.Error(), nil)
		return
	}
	file.Close()
	s.sendResponse(true, "File created successfully", nil)
}

func (s *Session) handleGetFile(payload json.RawMessage) {
	type GetFile struct {
		FilePath string `json:"filePath"`
	}
	var data GetFile
	json.Unmarshal(payload, &data)

	if _, err := os.Stat(data.FilePath); os.IsNotExist(err) {
		s.sendResponse(false, "File does not exist", nil)
		return
	}
	content, err := os.ReadFile(data.FilePath)
	if err != nil {
		s.sendResponse(false, "Error reading file", nil)
		return
	}
	type FileContent struct {
		Content string `json:"content"`
	}
	resp, _ := json.Marshal(FileContent{Content: string(content)})
	s.sendResponse(true, "File retrieved successfully", resp)
}

func (s *Session) handleUpdateFile(payload json.RawMessage) {
	type UpdateFile struct {
		FilePath string `json:"filePath"`
		Content  string `json:"content"`
	}
	var data UpdateFile
	json.Unmarshal(payload, &data)

	decoded, err := base64.StdEncoding.DecodeString(data.Content)
	if err != nil {
		s.sendResponse(false, "Error decoding content", nil)
		return
	}

	// Secure permissions
	if err := os.WriteFile(data.FilePath, decoded, 0644); err != nil {
		s.sendResponse(false, "Error writing file", nil)
		return
	}
	s.sendResponse(true, "File updated successfully", nil)
}

func (s *Session) handleDeleteFile(payload json.RawMessage) {
	var data struct {
		FileName string `json:"fileName"`
		FilePath string `json:"filePath"`
	}
	json.Unmarshal(payload, &data)
	os.Remove(filepath.Join(data.FilePath, data.FileName))
	s.sendResponse(true, "File deleted successfully", nil)
}

func (s *Session) handleCreateFolder(payload json.RawMessage) {
	var data struct {
		FolderName string `json:"folderName"`
		FolderPath string `json:"folderPath"`
	}
	json.Unmarshal(payload, &data)
	os.MkdirAll(filepath.Join(data.FolderPath, data.FolderName), 0755)
	s.sendResponse(true, "Folder created successfully", nil)
}

func (s *Session) handleDeleteFolder(payload json.RawMessage) {
	var data struct {
		FolderPath string `json:"folderPath"`
	}
	json.Unmarshal(payload, &data)
	os.RemoveAll(data.FolderPath)
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
	podName, err := s.K8sClient.WaitForPodByLabel(ctx, "cloud-ide", fmt.Sprintf("workspace=%s", s.WorkspaceID), 1*time.Second)
	if err != nil {
		s.sendResponse(false, "Error finding pod", nil)
		return
	}

	wsWriter := &WSWriter{Conn: s.Conn}
	cmd := []string{"bin/bash", "-c", data.Instruction}
	s.K8sClient.ExecToPod(ctx, "cloud-ide", podName, "shell", cmd, nil, wsWriter, wsWriter, false)
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

	// Restore session state if empty (e.g. page refresh)
	if s.WorkspaceID == "" {
		s.WorkspaceID = targetId
		// Ideally we restore s.K8sClient here too if needed
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
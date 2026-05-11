package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type CommandRequest struct {
	Command string `json:"command"`
	Param   string `json:"param"`
}

// saveDir is where uploaded files are stored on the slave machine.
// Change this to any absolute path you prefer, e.g. "C:\\Received".
const saveDir = "received"

func executeCommand(cmd CommandRequest) error {
	switch cmd.Command {
	case "background":
		script := fmt.Sprintf(`
			$code = @"
			using System.Runtime.InteropServices;
			public class Wallpaper {
				[DllImport("user32.dll")]
				public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
			}
"@
			Add-Type $code
			[Wallpaper]::SystemParametersInfo(20, 0, "%s", 3)
		`, cmd.Param)
		return exec.Command("powershell", "-Command", script).Run()
	case "lock":
		return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
	case "shutdown":
		return exec.Command("shutdown", "/s", "/t", "0").Run()
	default:
		return fmt.Errorf("unknown command: %s", cmd.Command)
	}
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received command: %s (param: %s)\n", cmd.Command, cmd.Param)

	if err := executeCommand(cmd); err != nil {
		log.Printf("Error executing command: %v\n", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Command executed successfully")
}

// uploadHandler receives a file sent by the master and saves it to saveDir.
// The master sends the file as multipart/form-data with the field name "file".
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit upload size to 512 MB to avoid memory exhaustion.
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing 'file' field: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Ensure the destination directory exists.
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "Cannot create save directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Use only the base name to prevent path-traversal attacks.
	safeName := filepath.Base(header.Filename)
	destPath := filepath.Join(saveDir, safeName)

	dest, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Cannot create file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	written, err := io.Copy(dest, file)
	if err != nil {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Received file '%s' (%d bytes) → saved to %s\n", safeName, written, destPath)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Saved '%s' (%d bytes)\n", safeName, written)
}

func main() {
	// Ensure the receive directory exists at startup.
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		log.Fatalf("Cannot create save directory: %v\n", err)
	}

	http.HandleFunc("/command", commandHandler)
	http.HandleFunc("/upload", uploadHandler)

	port := ":8080"
	log.Printf("Slave listening on port %s …\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

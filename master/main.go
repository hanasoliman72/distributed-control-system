package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type CommandRequest struct {
	Command   string `json:"command"`
	Parameter string `json:"param"`
}

var slaves = []string{
	"http://127.0.0.1:8080",
}

// sendTo sends a JSON command to a slave machine.
func sendTo(url string, cmd CommandRequest) {
	body, _ := json.Marshal(cmd)
	resp, err := http.Post(url+"/command", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Failed to reach slave %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Slave %s responded: %s\n", url, resp.Status)
}

// sendFileTo uploads a file to a single slave using multipart/form-data.
// The slave will save the file under the same filename in its working directory.
func sendFileTo(slaveURL, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Cannot open file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	// Build a multipart body with one field named "file".
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		log.Printf("Failed to create form file: %v\n", err)
		return
	}
	if _, err = io.Copy(part, file); err != nil {
		log.Printf("Failed to copy file data: %v\n", err)
		return
	}
	writer.Close() // Must be called before the request is sent to finalise the boundary.

	resp, err := http.Post(slaveURL+"/upload", writer.FormDataContentType(), &buf)
	if err != nil {
		log.Printf("Failed to upload to slave %s: %v\n", slaveURL, err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Slave %s upload response (%s): %s\n", slaveURL, resp.Status, respBody)
}

func executeLocally(cmd CommandRequest) {
	var err error
	switch cmd.Command {
	case "background":
		script := fmt.Sprintf(`
			$code = @"
			using System.Runtime.InteropServices;
			public class Wallpaper {
				[DllImport("user32.dll")]
				public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
			}"@
			Add-Type $code
			[Wallpaper]::SystemParametersInfo(20, 0, "%s", 3)
		`, cmd.Parameter)
		err = exec.Command("powershell", "-Command", script).Run()
	case "lock":
		err = exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
	case "shutdown":
		err = exec.Command("shutdown", "/s", "/t", "0").Run()
	}
	if err != nil {
		log.Printf("❌ Local execution failed: %v\n", err)
	} else {
		log.Println("✅ Local command executed")
	}
}

// broadcast sends a command to every slave and also runs it locally.
func broadcast(cmd CommandRequest) {
	for _, slaveURL := range slaves {
		sendTo(slaveURL, cmd)
	}
	executeLocally(cmd)
	log.Println("Broadcast complete.")
}

// broadcastFile sends a file to every slave machine.
// It does NOT copy the file locally — the master already has it.
func broadcastFile(filePath string) {
	for _, slaveURL := range slaves {
		log.Printf("Sending %s to %s …\n", filePath, slaveURL)
		sendFileTo(slaveURL, filePath)
	}
	log.Println("File broadcast complete.")
}

func main() {
	var choice int
	var param string

	for {
		fmt.Println("\n=== MASTER CONTROL ===")
		fmt.Println("1. Change Desktop Background")
		fmt.Println("2. Lock all PCs")
		fmt.Println("3. Shutdown all PCs")
		fmt.Println("4. Send File to all PCs")
		fmt.Println("0. Exit")
		fmt.Print("Choose: ")
		fmt.Scan(&choice)

		switch choice {
		case 1:
			fmt.Print("Enter full image path (e.g. C:\\images\\bg.jpg): ")
			fmt.Scan(&param)
			broadcast(CommandRequest{Command: "background", Parameter: param})
		case 2:
			broadcast(CommandRequest{Command: "lock"})
		case 3:
			broadcast(CommandRequest{Command: "shutdown"})
		case 4:
			fmt.Print("Enter full path of the file to send (e.g. C:\\files\\doc.pdf): ")
			fmt.Scan(&param)
			broadcastFile(param)
		case 0:
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

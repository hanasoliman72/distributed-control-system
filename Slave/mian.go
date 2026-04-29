package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

type CommandRequest struct {
	Command string `json:"command"` 
	Param   string `json:"param"`   
}

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
		// Shuts down immediately
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

func main() {
	http.HandleFunc("/command", commandHandler)

	port := ":8080"
	log.Printf("Slave listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
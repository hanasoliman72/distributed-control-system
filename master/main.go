package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

type CommandRequest struct {
	Command   string `json:"command"`
	Parameter string `json:"param"`
}

var slaves = []string{
	"http://172.20.10.2:8080",
}

func sendTo(url string, cmd CommandRequest) {
	body, _ := json.Marshal(cmd) // Converts the CommandRequest struct into JSON bytes.
	resp, err := http.Post(url+"/command", "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("Failed to reach slave %s: %v\n", url, err)
		return
	}

	defer resp.Body.Close() // Close the http connection
	log.Printf("Slave %s responded: %s\n", url, resp.Status)
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
		err = exec.Command("shutdown", "/s", "/t", "0").Run() // /s = shutdown, /t 0 = after 0 seconds.
	}

	if err != nil {
		log.Printf("❌ Local execution failed: %v\n", err)
	} else {
		log.Println("✅ Local command executed")
	}
}

func broadcast(cmd CommandRequest) {
	for _, slaveURL := range slaves {
		sendTo(slaveURL, cmd)
	}

	executeLocally(cmd)

	log.Println("Broadcast complete.")
}

func main() {
	var choice int
	var param string

	for {
		fmt.Println("\n=== MASTER CONTROL ===")
		fmt.Println("1. Change Desktop Background")
		fmt.Println("2. Lock all PCs")
		fmt.Println("3. Shutdown all PCs")
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
		case 0:
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

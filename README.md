# Distributed Command & Control System 🖥️

A distributed master-slave system built in Go where a master node remotely
executes system commands across multiple slave nodes via HTTP REST API.

Built as part of the **Distributed Database (DDB) course**.

## 📌 Features

- 🖼️ Change desktop wallpaper on all machines
- 🔒 Lock all machines simultaneously
- ⚡ Shutdown all machines simultaneously
- 🌐 HTTP REST API communication
- 🖥️ Master executes commands locally too

## 📁 Project Structure

distributed-control-system/
├── master/
│   └── main.go       # Master node - sends commands
├── slave/
│   └── main.go       # Slave node - receives & executes commands
└── README.md

## 🚀 How to Run

### Prerequisites
- [Go](https://golang.org/dl/) installed on all machines
- All machines on the **same network**
- Port **8080** open on slave machines

### On each Slave PC:
```bash
go run slave/main.go
```

### On the Master PC:
```bash
go run master/main.go
```

## ⚙️ Configuration

In `master/main.go`, update the slave IPs to match your network:

```go
var slaves = []string{
    "http://192.168.1.101:8080",
    "http://192.168.1.102:8080",
    "http://192.168.1.103:8080",
    "http://192.168.1.104:8080",
}
```

## 🛡️ Firewall Setup (Windows)

Run this on each slave machine to allow incoming connections:

```bash
netsh advfirewall firewall add rule name="Go Slave" dir=in action=allow protocol=TCP localport=8080
```

## 🧰 Technologies Used

- **Go** - Main programming language
- **net/http** - HTTP server & client
- **encoding/json** - JSON serialization
- **os/exec** - System command execution
- **PowerShell** - Windows wallpaper API
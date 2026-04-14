package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command(".\\orquestador-auditor.exe", "--mcp")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	b, _ := json.Marshal(req)
	fmt.Fprintf(stdin, "%s\n", b)

	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		fmt.Println("RESPONSE:", scanner.Text())
	}
	stdin.Close()
	cmd.Wait()
}

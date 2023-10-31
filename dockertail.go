package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fatih/color"
)

func main() {
	var follow bool
	var lines int

	flag.BoolVar(&follow, "f", false, "Show logs in real-time")
	flag.IntVar(&lines, "n", 0, "Show the last n lines of logs")
	flag.Parse()

	containerIDs := flag.Args()

	if len(containerIDs) == 0 {
		fmt.Println("Please provide at least one container ID")
		return
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	colors := []func(a ...interface{}) string{
		color.New(color.FgGreen).SprintFunc(),
		color.New(color.FgYellow).SprintFunc(),
		color.New(color.FgBlue).SprintFunc(),
		color.New(color.FgMagenta).SprintFunc(),
		color.New(color.FgCyan).SprintFunc(),
		color.New(color.FgRed).SprintFunc(),
	}

	colorMap := make(map[string]func(a ...interface{}) string)

	for i, id := range containerIDs {
		colorMap[id] = colors[i%len(colors)]
	}

	for _, containerID := range containerIDs {
		go func(containerID string) {
			options := types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
			}

			if follow {
				options.Follow = true
			}

			if lines > 0 {
				options.Tail = fmt.Sprintf("%d", lines)
			}

			out, err := cli.ContainerLogs(context.Background(), containerID, options)
			if err != nil {
				fmt.Printf("Error retrieving logs for container %s: %s\n", containerID, err)
				return
			}

			defer out.Close()

			fmt.Printf("Logs for container: %s\n", containerID)
			colorFunc, exists := colorMap[containerID]
			if !exists {
				colorFunc = color.New(color.FgWhite).SprintFunc()
			}

			scanner := bufio.NewScanner(out)
			for scanner.Scan() {
				fmt.Println(colorFunc(scanner.Text()))
			}
		}(containerID)
	}

	select {}
}

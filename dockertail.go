package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fatih/color"
)

// Function to remove non-printable characters from a string
func cleanLogLine(input string) string {
	// This regex will match and remove all non-printable characters
	reg, err := regexp.Compile("[\x00-\x1F\x7F]")
	if err != nil {
		fmt.Printf("Error compiling regex: %s\n", err)
		return input
	}
	return reg.ReplaceAllString(input, "")
}

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
	nameMap := make(map[string]string)

	for i, id := range containerIDs {
		colorMap[id] = colors[i%len(colors)]

		containerJSON, err := cli.ContainerInspect(context.Background(), id)
		if err != nil {
			fmt.Printf("Error inspecting container %s: %s\n", id, err)
			return
		}
		nameMap[id] = containerJSON.Name[1:] // Remove leading '/'
	}

	// Colors for log levels
	errorColor := color.New(color.FgRed, color.Bold).SprintFunc()
	warnColor := color.New(color.FgYellow, color.Bold).SprintFunc()
	infoColor := color.New(color.FgWhite).SprintFunc()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for _, containerID := range containerIDs {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
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

			out, err := cli.ContainerLogs(ctx, containerID, options)
			if err != nil {
				fmt.Printf("Error retrieving logs for container %s: %s\n", containerID, err)
				return
			}
			defer out.Close()

			colorFunc, exists := colorMap[containerID]
			if !exists {
				colorFunc = color.New(color.FgWhite).SprintFunc()
			}
			containerName := nameMap[containerID]

			scanner := bufio.NewScanner(out)
			for scanner.Scan() {
				logLine := scanner.Text()
				logLine = cleanLogLine(logLine) // Clean the log line

				// Determine log level and apply corresponding color
				if strings.Contains(logLine, "ERROR") {
					logLine = errorColor(logLine)
				} else if strings.Contains(logLine, "WARN") {
					logLine = warnColor(logLine)
				} else {
					logLine = infoColor(logLine)
				}

				timestamp := time.Now().Format("2006-01-02 15:04:05")
				fmt.Printf("%s [%s] %s\n", timestamp, colorFunc(containerName), logLine)
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading logs from container %s: %s\n", containerID, err)
			}
		}(containerID)
	}

	// Setup signal catching
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals // Wait for a termination signal
	cancel()  // Cancel the context

	wg.Wait() // Wait for all goroutines to finish
}

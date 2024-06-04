package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
    "fyne.io/fyne/v2/desktop"
)

const (
	dockerNetwork = "my_network"
	linkPath      = "/usr/local/bin/php"
)

func createDockerNetwork() error {
	cmd := exec.Command("docker", "network", "inspect", dockerNetwork)
	if err := cmd.Run(); err != nil {
		cmd := exec.Command("docker", "network", "create", dockerNetwork)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create docker network: %v", err)
		}
	}
	return nil
}

func createDockerVolume(volumeName string) error {
	cmd := exec.Command("docker", "volume", "inspect", volumeName)
	if err := cmd.Run(); err != nil {
		cmd := exec.Command("docker", "volume", "create", volumeName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create docker volume: %v", err)
		}
	}
	return nil
}

func startMariaDBContainer(label *widget.Label) error {
	label.SetText("Starting MariaDB container...")

	volumeName := "php-gui-mariadb_data"
	if err := createDockerVolume(volumeName); err != nil {
		return fmt.Errorf("failed to create Docker volume: %v", err)
	}

	mariadbCommand := fmt.Sprintf("docker run --rm -d -p 3306:3306 -v %s:/var/lib/mysql --network %s --name mariadb -e MYSQL_ROOT_PASSWORD=root mariadb",
		volumeName, dockerNetwork)
	if err := exec.Command("sh", "-c", mariadbCommand).Run(); err != nil {
		return fmt.Errorf("failed to start MariaDB container: %v", err)
	}
	label.SetText("MariaDB started.")
	return nil
}

func stopMariaDBContainer() {
	exec.Command("docker", "stop", "mariadb").Run()
}

func stopPHPContainers() {
	exec.Command("docker", "stop", "custom-php").Run()
	exec.Command("docker", "stop", "dunglas/frankenphp:1-php8.2").Run()
	exec.Command("docker", "stop", "dunglas/frankenphp:1-php8.3").Run()
}

func buildCustomDockerImage(dockerfilePath string, label *widget.Label) (string, error) {
	imageName := "custom-php:latest"
	label.SetText("Building custom Docker image...")
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath, filepath.Dir(dockerfilePath))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build custom docker image: %v, output: %s", err, output)
	}
	label.SetText("Custom Docker image built.")
	return imageName, nil
}

func createPHPCommandScript(imageName string) error {
	// Remove existing symbolic link or file
	if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing symlink: %v", err)
	}

	// Create new symbolic link
	phpCommand := fmt.Sprintf("docker run --rm --network %s -v $PWD:/app/public %s php \"$@\"", dockerNetwork, imageName)
	scriptContent := fmt.Sprintf("#!/bin/sh\n%s\n", phpCommand)
	if err := os.WriteFile(linkPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write script: %v", err)
	}

	return nil
}

func runPHPContainer(version string, label *widget.Label) error {
	imageName := fmt.Sprintf("dunglas/frankenphp:1-php%s", version)

	// Check if the image already exists
	checkCmd := exec.Command("docker", "images", "-q", imageName)
	checkOutput, _ := checkCmd.Output()

	// If the image does not exist, pull it
	if len(checkOutput) == 0 {
		label.SetText(fmt.Sprintf("Pulling PHP %s Docker image...", version))
		pullCmd := exec.Command("docker", "pull", imageName)
		if output, err := pullCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to pull Docker image: %v, output: %s", err, output)
		}
		label.SetText(fmt.Sprintf("PHP %s Docker image pulled.", version))
	} else {
		label.SetText(fmt.Sprintf("PHP %s Docker image already exists.", version))
	}

	return createPHPCommandScript(imageName)
}

func runCustomContainer(dockerfilePath string, label *widget.Label) error {
	imageName, err := buildCustomDockerImage(dockerfilePath, label)
	if err != nil {
		return err
	}
	return createPHPCommandScript(imageName)
}

func handleQuit(myApp fyne.App, myWindow fyne.Window) {
	fmt.Println("Quitting...")

	dialog := dialog.NewConfirm("Quit", "Are you sure you want to quit?", func(confirmed bool) {
		if confirmed {
			stopMariaDBContainer()
			stopPHPContainers()
			myApp.Quit()
		}
	}, myWindow)

	dialog.Show()
}

func main() {
	// Run in detached mode if called with "detached" argument
	if len(os.Args) > 1 && os.Args[1] == "detached" {
		cmd := exec.Command(os.Args[0])
		cmd.Start()
		fmt.Println("Running in detached mode. PID:", cmd.Process.Pid)
		return
	}

	myApp := app.New()
	myWindow := myApp.NewWindow("PHP & MariaDB Version Switcher")

	if desk, ok := myApp.(desktop.App); ok {
		myMenu := fyne.NewMenu("MyApp",
			fyne.NewMenuItem("Show", func() {
				myWindow.Show()
			}),
			fyne.NewMenuItem("Quit", func() {
				handleQuit(myApp, myWindow)
			}),
		)
		desk.SetSystemTrayMenu(myMenu)
	}

	label := widget.NewLabel("Select PHP Version:")

	buttonMariaDB := widget.NewButton("Start MariaDB", func() {
		if err := createDockerNetwork(); err != nil {
			label.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		go func() {
			if err := startMariaDBContainer(label); err != nil {
				label.SetText(fmt.Sprintf("Error: %v", err))
			} else {
				label.SetText("MariaDB started.")
			}
		}()
	})

	buttonPHP82 := widget.NewButton("PHP 8.2", func() {
		if err := createDockerNetwork(); err != nil {
			label.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		go func() {
			if err := runPHPContainer("8.2", label); err != nil {
				label.SetText(fmt.Sprintf("Error: %v", err))
			} else {
				label.SetText("PHP 8.2 set.")
			}
		}()
	})

	buttonPHP83 := widget.NewButton("PHP 8.3", func() {
		if err := createDockerNetwork(); err != nil {
			label.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		go func() {
			if err := runPHPContainer("8.3", label); err != nil {
				label.SetText(fmt.Sprintf("Error: %v", err))
			} else {
				label.SetText("PHP 8.3 set.")
			}
		}()
	})

	buttonCustom := widget.NewButton("Custom", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				if err := createDockerNetwork(); err != nil {
					label.SetText(fmt.Sprintf("Error: %v", err))
					return
				}
				go func() {
					if err := runCustomContainer(reader.URI().Path(), label); err != nil {
						label.SetText(fmt.Sprintf("Error: %v", err))
					} else {
						label.SetText("Custom PHP set.")
					}
				}()
			}
		}, myWindow)
	})

	myWindow.SetContent(container.NewVBox(
		label,
		buttonMariaDB,
		buttonPHP82,
		buttonPHP83,
		buttonCustom,
	))

	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})
	myWindow.ShowAndRun()

	// Handle application close event to stop containers
	myWindow.SetOnClosed(func() {
		stopMariaDBContainer()
		// Add other cleanup tasks here if necessary
	})

	// Handle system signals for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		stopMariaDBContainer()
		stopPHPContainers()
		// Add other cleanup tasks here if necessary
		os.Exit(0)
	}()

	myWindow.ShowAndRun()
}

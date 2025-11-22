package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// ExportImages exports the selected Docker images to a local destination
func ExportImages(destination string) {
	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("[x] Failed to create Docker client: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	// List Docker images
	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		fmt.Printf("[x] Failed to list Docker images: %v\n", err)
		os.Exit(1)
	}

	if len(images) == 0 {
		fmt.Println("[x] No Docker images found")
		os.Exit(1)
	}

	// Format image names for selection
	imageNames := make([]string, 0, len(images))
	for _, img := range images {
		for _, tag := range img.RepoTags {
			// Skip <none>:<none> tags
			if tag != "<none>:<none>" {
				// If grep pattern is provided, only add images that match the pattern
				if os.Getenv("DKCI_GREP_PATTERN") != "" { // Using env var to pass grep pattern
					if strings.Contains(tag, os.Getenv("DKCI_GREP_PATTERN")) {
						imageNames = append(imageNames, tag)
					}
				} else {
					imageNames = append(imageNames, tag)
				}
			}
		}
	}

	if len(imageNames) == 0 {
		fmt.Println("[x] No tagged Docker images found")
		os.Exit(1)
	}

	fmt.Printf("Found %d tagged Docker image(s)\n", len(imageNames))

	// Setup multi-select options
	selections := []string{}

	// Add an "All" option if there are multiple images
	if len(imageNames) > 1 {
		selections = append([]string{"All"}, imageNames...)
	} else {
		selections = imageNames
	}

	// Multi-select prompt
	prompt := &survey.MultiSelect{
		Message: "Select Docker images to export:",
		Options: selections,
	}

	selectedImages := []string{}
	err = survey.AskOne(prompt, &selectedImages)
	if err != nil {
		fmt.Printf("[x] Failed to get user selection: %v\n", err)
		os.Exit(1)
	}

	// Handle the "All" selection
	if len(selectedImages) == 1 && selectedImages[0] == "All" {
		selectedImages = imageNames // Select all images
	}

	if len(selectedImages) == 0 {
		fmt.Println("[x] No images selected")
		os.Exit(1)
	}

	fmt.Printf("Selected images: %v\n", selectedImages)

	// Create destination directory if it doesn't exist
	err = os.MkdirAll(destination, 0755)
	if err != nil {
		fmt.Printf("[x] Failed to create destination directory %s: %v\n", destination, err)
		os.Exit(1)
	}

	// Export selected images
	for _, imageName := range selectedImages {
		ExportImage(cli, imageName, destination)
	}
}

func ExportImage(cli *client.Client, imageName, destination string) {
	// Inspect the image to get additional info like OS and architecture
	imageInspect, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
	var osInfo, archInfo string
	if err != nil {
		// If inspection fails, we'll use empty values for OS and arch, but log the error
		fmt.Printf("Warning: Could not inspect image %s: %v\n", imageName, err)
		osInfo = ""
		archInfo = ""
	} else {
		osInfo = imageInspect.Os
		archInfo = imageInspect.Architecture
	}

	// Parse the image name and tag
	nameParts := strings.Split(imageName, ":")
	imageNameOnly := nameParts[0]
	tag := ""
	if len(nameParts) > 1 {
		tag = nameParts[1]
	}

	// Sanitize the image name for filename (replace '/' with '·')
	sanitizedImageName := strings.ReplaceAll(imageNameOnly, "/", "·")

	var tarFileName string
	// Format: <image_name>_<tag>_<os>_<arch>.tar
	var suffixParts []string
	if tag != "" {
		suffixParts = append(suffixParts, tag)
	} else {
		// Always include a tag value, use "latest" as default if not available
		suffixParts = append(suffixParts, "latest")
	}
	if osInfo != "" {
		suffixParts = append(suffixParts, osInfo)
	} else {
		// If OS info is not available, add a placeholder
		suffixParts = append(suffixParts, "unknown")
	}
	if archInfo != "" {
		suffixParts = append(suffixParts, archInfo)
	} else {
		// If architecture info is not available, add a placeholder
		suffixParts = append(suffixParts, "unknown")
	}

	tarFileName = fmt.Sprintf("%s_%s.tar", sanitizedImageName, strings.Join(suffixParts, "_"))

	tarFilePath := filepath.Join(destination, tarFileName)

	fmt.Printf("Exporting image %s to %s...\n", imageName, tarFilePath)

	// Export the image
	imageReader, err := cli.ImageSave(context.Background(), []string{imageName})
	if err != nil {
		fmt.Printf("[x] Failed to export image %s: %v\n", imageName, err)
		return
	}
	defer imageReader.Close()

	// Create the output file
	outFile, err := os.Create(tarFilePath)
	if err != nil {
		fmt.Printf("[x] Failed to create output file %s: %v\n", tarFilePath, err)
		return
	}
	defer outFile.Close()

	// Copy the image data to the tar file
	_, err = io.Copy(outFile, imageReader)
	if err != nil {
		fmt.Printf("[x] Failed to write image %s to file %s: %v\n", imageName, tarFilePath, err)
		return
	}

	fmt.Printf("[√] Successfully exported image %s to %s\n", imageName, tarFilePath)
}

// DeleteImages deletes the selected Docker images
func DeleteImages(grepPattern string) {
	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("[x] Failed to create Docker client: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	// List Docker images
	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		fmt.Printf("[x] Failed to list Docker images: %v\n", err)
		os.Exit(1)
	}

	if len(images) == 0 {
		fmt.Println("[x] No Docker images found")
		os.Exit(1)
	}

	// Format image names for selection
	imageNames := make([]string, 0, len(images))
	for _, img := range images {
		for _, tag := range img.RepoTags {
			// Skip <none>:<none> tags
			if tag != "<none>:<none>" {
				// If grep pattern is provided, only add images that match the pattern
				if grepPattern != "" {
					if strings.Contains(tag, grepPattern) {
						imageNames = append(imageNames, tag)
					}
				} else {
					imageNames = append(imageNames, tag)
				}
			}
		}
	}

	if len(imageNames) == 0 {
		fmt.Println("[x] No tagged Docker images found")
		os.Exit(1)
	}

	fmt.Printf("Found %d tagged Docker image(s)\n", len(imageNames))

	// Setup multi-select options
	selections := []string{}

	// Add an "All" option if there are multiple images
	if len(imageNames) > 1 {
		selections = append([]string{"All"}, imageNames...)
	} else {
		selections = imageNames
	}

	// Multi-select prompt
	prompt := &survey.MultiSelect{
		Message: "Select Docker images to delete:",
		Options: selections,
	}

	selectedImages := []string{}
	err = survey.AskOne(prompt, &selectedImages)
	if err != nil {
		fmt.Printf("[x] Failed to get user selection: %v\n", err)
		os.Exit(1)
	}

	// Handle the "All" selection
	if len(selectedImages) == 1 && selectedImages[0] == "All" {
		selectedImages = imageNames // Select all images
	}

	if len(selectedImages) == 0 {
		fmt.Println("[x] No images selected")
		os.Exit(1)
	}

	fmt.Printf("Selected images: %v\n", selectedImages)

	// Delete selected images
	for _, imageName := range selectedImages {
		DeleteImage(cli, imageName)
	}
}

func DeleteImage(cli *client.Client, imageName string) {
	fmt.Printf("Deleting image %s...\n", imageName)

	// Delete the image
	_, err := cli.ImageRemove(context.Background(), imageName, types.ImageRemoveOptions{
		Force:         false, // Don't force deletion by default
		PruneChildren: true,  // Remove dependent images too
	})
	if err != nil {
		fmt.Printf("[x] Failed to delete image %s: %v\n", imageName, err)
		return
	}

	fmt.Printf("[√] Successfully deleted image %s\n", imageName)
}

// CleanCache deletes all files in the cache directory
func CleanCache() {
	cacheDir := "/tmp/go-dkci"

	// Check if directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Printf("[x] Cache directory does not exist: %s\n", cacheDir)
		os.Exit(1)
	}

	// Read all files in the directory
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		fmt.Printf("[x] Failed to read cache directory %s: %v\n", cacheDir, err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No files found in cache directory: %s\n", cacheDir)
		return
	}

	// List and count files to be deleted
	var filesToDelete []string
	for _, file := range files {
		filePath := filepath.Join(cacheDir, file.Name())
		filesToDelete = append(filesToDelete, filePath)
		fmt.Printf("- %s\n", filePath)
	}

	// Confirm deletion with user
	fmt.Printf("\nFound %d file(s) in cache directory. Are you sure you want to delete all?\n", len(filesToDelete))

	// Simple confirmation - in a real app we might want to use a proper confirmation prompt
	confirmed := false
	fmt.Print("Type 'yes' to confirm deletion: ")
	var response string
	fmt.Scanln(&response)
	if response == "yes" {
		confirmed = true
	}

	if !confirmed {
		fmt.Println("[x] Cache cleanup cancelled by user")
		return
	}

	// Delete all files
	deletedCount := 0
	for _, filePath := range filesToDelete {
		if err := os.RemoveAll(filePath); err != nil {
			fmt.Printf("[x] Failed to delete %s: %v\n", filePath, err)
		} else {
			deletedCount++
		}
	}

	fmt.Printf("[√] Successfully cleaned cache directory. Deleted %d file(s)\n", deletedCount)
}

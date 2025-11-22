package cloud

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/baowuhe/go-bdfs/pan"
	"github.com/baowuhe/go-dkci/config"
	"github.com/baowuhe/go-dkci/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// ExportImagesToCloud exports the selected Docker images to Baidu cloud disk
func ExportImagesToCloud(cloudPath string) {
	// Get BDFS configuration
	configData, err := config.GetBDFSConfig()
	if err != nil {
		fmt.Printf("[x] Error getting BDFS configuration: %v\n", err)
		os.Exit(1)
	}

	// Create a BDFS client with the provided config
	bdfsClient := pan.NewClient(configData.ClientID, configData.ClientSecret, configData.TokenPath)

	// Login to Baidu cloud
	if err := bdfsClient.Authorize(context.Background()); err != nil {
		fmt.Printf("[x] Failed to login to Baidu cloud: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[√] Successfully logged in to Baidu cloud")

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
		Message: "Select Docker images to export to cloud:",
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

	// Export selected images to cloud
	for _, imageName := range selectedImages {
		ExportImageToCloud(cli, imageName, cloudPath, bdfsClient)
	}
}

func ExportImageToCloud(cli *client.Client, imageName, cloudPath string, bdfsClient *pan.Client) {
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

	// Create temporary file to save the image
	tempDir := "/tmp/go-dkci"
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		fmt.Printf("[x] Failed to create temp directory %s: %v\n", tempDir, err)
		return
	}

	tempFilePath := filepath.Join(tempDir, tarFileName)

	fmt.Printf("Exporting image %s to temporary file %s...\n", imageName, tempFilePath)

	// Export the image to temporary file
	imageReader, err := cli.ImageSave(context.Background(), []string{imageName})
	if err != nil {
		fmt.Printf("[x] Failed to export image %s: %v\n", imageName, err)
		return
	}
	defer imageReader.Close()

	// Create the output file
	outFile, err := os.Create(tempFilePath)
	if err != nil {
		fmt.Printf("[x] Failed to create temporary file %s: %v\n", tempFilePath, err)
		return
	}
	defer outFile.Close()

	// Copy the image data to the temporary tar file
	_, err = io.Copy(outFile, imageReader)
	if err != nil {
		fmt.Printf("[x] Failed to write image %s to temporary file %s: %v\n", imageName, tempFilePath, err)
		return
	}

	// Upload the temporary file to Baidu cloud
	remoteFilePath := filepath.Join(cloudPath, tarFileName)

	fmt.Printf("Uploading %s to Baidu cloud path %s...\n", tempFilePath, remoteFilePath)
	if err := bdfsClient.UploadFile(tempFilePath, remoteFilePath); err != nil {
		fmt.Printf("[x] Failed to upload %s to Baidu cloud: %v\n", tempFilePath, err)
		// Clean up the temporary file
		os.Remove(tempFilePath)
		return
	}

	// Clean up the temporary file after successful upload
	if err := os.Remove(tempFilePath); err != nil {
		fmt.Printf("Warning: Failed to remove temporary file %s: %v\n", tempFilePath, err)
	}

	fmt.Printf("[√] Successfully exported and uploaded image %s to %s\n", imageName, remoteFilePath)
}

// ImportImagesFromCloud downloads Docker images from Baidu cloud disk and imports them to local Docker
func ImportImagesFromCloud(cloudPath string, grepPattern string) {
	// Get BDFS configuration
	configData, err := config.GetBDFSConfig()
	if err != nil {
		fmt.Printf("[x] Error getting BDFS configuration: %v\n", err)
		os.Exit(1)
	}

	// Create a BDFS client with the provided config
	bdfsClient := pan.NewClient(configData.ClientID, configData.ClientSecret, configData.TokenPath)

	// Login to Baidu cloud
	if err := bdfsClient.Authorize(context.Background()); err != nil {
		fmt.Printf("[x] Failed to login to Baidu cloud: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[√] Successfully logged in to Baidu cloud")

	// Check if the cloud path is a directory by trying to list it
	files, err := bdfsClient.ListFiles(cloudPath)
	if err != nil {
		// If listing fails, assume it's a single file
		// Check if it's a tar file
		fileInfo, err := bdfsClient.GetFileInfoByPath(cloudPath)
		if err != nil {
			fmt.Printf("[x] Error accessing cloud file %s: %v\n", cloudPath, err)
			os.Exit(1)
		}

		if strings.HasSuffix(strings.ToLower(fileInfo.Path), ".tar") ||
			strings.HasSuffix(strings.ToLower(fileInfo.Path), ".tar.gz") ||
			strings.HasSuffix(strings.ToLower(fileInfo.Path), ".tgz") {

			// Directly download and import the single file
			downloadAndImportFromCloud(bdfsClient, fileInfo.Path)
		} else {
			// The path is a file but not a tar file
			fmt.Printf("[x] The specified file %s is not a .tar file\n", cloudPath)
			os.Exit(1)
		}
	} else {
		// It's a directory, filter files to only include .tar files
		tarFiles := []pan.FileInfo{}
		for _, file := range files {
			if strings.HasSuffix(strings.ToLower(file.Path), ".tar") ||
				strings.HasSuffix(strings.ToLower(file.Path), ".tar.gz") ||
				strings.HasSuffix(strings.ToLower(file.Path), ".tgz") {

				// Apply grep filter if pattern is provided
				if grepPattern != "" {
					// Extract image name information from the file name for filtering
					baseName := strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
					// If the file name (without extension) contains the grep pattern, include it
					if strings.Contains(baseName, grepPattern) {
						tarFiles = append(tarFiles, file)
					}
				} else {
					tarFiles = append(tarFiles, file)
				}
			}
		}

		if len(tarFiles) == 0 {
			fmt.Println("[x] No .tar files found in the specified cloud directory")
			os.Exit(1)
		}

		// Prepare options for selection
		selectionOptions := make([]string, len(tarFiles))
		for i, file := range tarFiles {
			selectionOptions[i] = filepath.Base(file.Path)
		}

		// Add "All" option if there are more than 1 files
		if len(tarFiles) > 1 {
			selectionOptions = append([]string{"All"}, selectionOptions...)
		}

		// Show multi-select list to the user
		selectedFiles := []string{}
		prompt := &survey.MultiSelect{
			Message: "Select .tar files to download and import as Docker images:",
			Options: selectionOptions,
		}

		err = survey.AskOne(prompt, &selectedFiles)
		if err != nil {
			fmt.Printf("[x] Failed to get user selection: %v\n", err)
			os.Exit(1)
		}

		// Handle "All" selection
		if len(selectedFiles) == 1 && selectedFiles[0] == "All" {
			// Select all tar files
			selectedFiles = []string{}
			for _, file := range tarFiles {
				selectedFiles = append(selectedFiles, filepath.Base(file.Path))
			}
		}

		if len(selectedFiles) == 0 {
			fmt.Println("[x] No files selected for import")
			os.Exit(1)
		}

		// Map selected filenames back to full paths
		selectedFilePaths := []string{}
		for _, selectedFile := range selectedFiles {
			for _, tarFile := range tarFiles {
				if filepath.Base(tarFile.Path) == selectedFile {
					selectedFilePaths = append(selectedFilePaths, tarFile.Path)
					break
				}
			}
		}

		// Download and import each selected file
		for _, filePath := range selectedFilePaths {
			downloadAndImportFromCloud(bdfsClient, filePath)
		}
	}
}

// downloadAndImportFromCloud downloads a file from cloud and imports it as a Docker image
func downloadAndImportFromCloud(bdfsClient *pan.Client, cloudFilePath string) {
	// Create temporary directory for downloads
	tempDir := "/tmp/go-dkci"
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		fmt.Printf("[x] Failed to create temp directory %s: %v\n", tempDir, err)
		os.Exit(1)
	}

	// Download the file to the temporary directory
	localFilePath := filepath.Join(tempDir, filepath.Base(cloudFilePath))

	fmt.Printf("Downloading %s from Baidu cloud to temporary file %s...\n", cloudFilePath, localFilePath)
	// Download file content as stream
	resp, err := bdfsClient.DownloadFile(cloudFilePath)
	if err != nil {
		fmt.Printf("[x] Failed to download %s from Baidu cloud: %v\n", cloudFilePath, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Create local file to write to
	outFile, err := os.Create(localFilePath)
	if err != nil {
		fmt.Printf("[x] Failed to create local file %s: %v\n", localFilePath, err)
		os.Exit(1)
	}
	defer outFile.Close()

	// Copy downloaded content to local file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		fmt.Printf("[x] Failed to write downloaded content to %s: %v\n", localFilePath, err)
		os.Exit(1)
	}

	// Import the downloaded file using the existing docker import functionality
	docker.ImportImagesFromSource(localFilePath, "") // No grep pattern needed for single file download

	// Clean up the temporary file after successful import
	if err := os.Remove(localFilePath); err != nil {
		fmt.Printf("Warning: Failed to remove temporary file %s: %v\n", localFilePath, err)
	}
}

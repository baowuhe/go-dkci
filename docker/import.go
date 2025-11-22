package docker

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/docker/docker/client"
)

// ImportImagesFromSource imports Docker images from a specified source file or directory
func ImportImagesFromSource(source string, grepPattern string) {
	// Check if the source is a file or directory
	fileInfo, err := os.Stat(source)
	if err != nil {
		fmt.Printf("[x] Error accessing source: %v\n", err)
		os.Exit(1)
	}

	if fileInfo.IsDir() {
		// Handle directory import
		importFromDirectory(source, grepPattern)
	} else {
		// Handle single file import
		importFromFile(source)
	}
}

func importFromDirectory(dirPath string, grepPattern string) {
	// Find all .tar files in the directory
	tarFiles, err := findTarFilesInDirectory(dirPath, grepPattern)
	if err != nil {
		fmt.Printf("[x] Error finding .tar files: %v\n", err)
		os.Exit(1)
	}

	if len(tarFiles) == 0 {
		fmt.Println("[x] No .tar files found in the specified directory")
		os.Exit(1)
	}

	// Prepare options for selection
	selectionOptions := make([]string, len(tarFiles))
	for i, file := range tarFiles {
		selectionOptions[i] = filepath.Base(file)
	}

	// Add "All" option if there are more than 1 files
	if len(tarFiles) > 1 {
		selectionOptions = append([]string{"All"}, selectionOptions...)
	}

	// Show multi-select list to the user
	selectedFiles := []string{}
	prompt := &survey.MultiSelect{
		Message: "Select .tar files to import as Docker images:",
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
		for _, file := range tarFiles {
			selectedFiles = append(selectedFiles[1:], filepath.Base(file)) // Replace "All" with actual files
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
			if filepath.Base(tarFile) == selectedFile {
				selectedFilePaths = append(selectedFilePaths, tarFile)
				break
			}
		}
	}

	// Import each selected file
	for _, filePath := range selectedFilePaths {
		importFromFile(filePath)
	}
}

func importFromFile(filePath string) {
	fmt.Printf("Importing image from file: %s\n", filePath)

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("[x] Failed to create Docker client: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	// Open the tar file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("[x] Failed to open file %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	// Check if file is compressed with gzip
	_, err = file.Stat()
	if err != nil {
		fmt.Printf("[x] Failed to get file info: %v\n", err)
		os.Exit(1)
	}

	var imageReader io.Reader
	if strings.HasSuffix(strings.ToLower(filePath), ".tar.gz") || strings.HasSuffix(strings.ToLower(filePath), ".tgz") {
		// Uncompress gzip
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			fmt.Printf("[x] Failed to create gzip reader: %v\n", err)
			os.Exit(1)
		}
		defer gzipReader.Close()
		imageReader = gzipReader
	} else {
		imageReader = file
	}

	// Import the image
	response, err := cli.ImageLoad(context.Background(), imageReader, true) // quiet = true
	if err != nil {
		fmt.Printf("[x] Failed to load image from %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer response.Body.Close()

	// Read and display the response
	_, err = io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("[x] Failed to read import response: %v\n", err)
		os.Exit(1)
	}

	// Try to parse the tar file to get image information
	imageInfo, err := getImageInfoFromTar(filePath)
	if err != nil {
		// If we can't determine the image name, just report success
		fmt.Printf("[√] Successfully imported image from %s\n", filePath)
	} else {
		fmt.Printf("[√] Successfully imported image from %s: %s\n", filePath, imageInfo)
	}
}

func findTarFilesInDirectory(dirPath string, grepPattern string) ([]string, error) {
	var tarFiles []string
	
	// Walk through the directory to find .tar files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			lowerName := strings.ToLower(info.Name())
			if strings.HasSuffix(lowerName, ".tar") || 
				strings.HasSuffix(lowerName, ".tar.gz") || 
				strings.HasSuffix(lowerName, ".tgz") {
				
				// Apply grep filter if pattern is provided
				if grepPattern != "" {
					// Extract image name information from the file name for filtering
					baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
					// If the file name (without extension) contains the grep pattern, include it
					if strings.Contains(baseName, grepPattern) {
						tarFiles = append(tarFiles, path)
					}
				} else {
					tarFiles = append(tarFiles, path)
				}
			}
		}
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return tarFiles, nil
}

func getImageInfoFromTar(tarPath string) (string, error) {
	// Open the tar file
	file, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Check if file is compressed with gzip
	var tarReader io.Reader
	if strings.HasSuffix(strings.ToLower(tarPath), ".tar.gz") || strings.HasSuffix(strings.ToLower(tarPath), ".tgz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return "", err
		}
		defer gzipReader.Close()
		tarReader = gzipReader
	} else {
		// Seek back to the beginning if not compressed
		file.Seek(0, 0)
		tarReader = file
	}

	// Create a tar reader
	tarReaderVar := tar.NewReader(tarReader)

	// Look for the manifest.json file in the tar archive
	var manifestContent []byte
	for {
		header, err := tarReaderVar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Name == "manifest.json" {
			manifestContent, err = io.ReadAll(tarReaderVar)
			if err != nil {
				return "", err
			}
			break
		}
	}

	// If we found manifest.json content, we could parse it to get image information
	// For now, we'll just return the file name as basic information
	if len(manifestContent) > 0 {
		return filepath.Base(tarPath), nil
	}

	return filepath.Base(tarPath), nil
}
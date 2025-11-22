# go-dkci API Documentation

This document provides API documentation for the go-dkci project, organized by package.

## Table of Contents
- [config package](#config-package)
- [docker package](#docker-package)
- [cloud package](#cloud-package)

## config package

### Type: BDFSConfig
```go
type BDFSConfig struct {
    ClientID        string `toml:"client_id"`
    ClientSecret    string `toml:"client_secret"`
    TokenPath       string `toml:"token_path"`
    DefaultCloudDir string `toml:"default_cloud_dir"`
}
```

Represents the configuration structure for Baidu cloud.

### Function: GetBDFSConfig
```go
func GetBDFSConfig() (*BDFSConfig, error)
```

Retrieves the BDFS configuration from environment variables or TOML file.

The function first checks for individual environment variables:
- `BDFS_CLIENT_ID`
- `BDFS_CLIENT_SECRET`
- `BDFS_TOKEN_PATH`
- `BDFS_DEFAULT_CLOUD_DIR` (optional)

If all required environment variables are provided, it uses them directly.

If individual variables aren't all set, it checks for a config file path via:
- `BDFS_CONFIG_FILE` environment variable

If no custom config file is specified, it uses the default path: `~/.local/app/dkci/config.toml`

If the `DefaultCloudDir` is not specified, it defaults to "/".

Returns a pointer to a BDFSConfig struct or an error if configuration is incomplete.

## docker package

### Function: ExportImages
```go
func ExportImages(destination string)
```

Exports the selected Docker images to a local destination.

This function:
1. Initializes a Docker client
2. Lists all Docker images
3. Filters images based on optional grep pattern (from environment variable DKCI_GREP_PATTERN)
4. Shows a multi-select prompt to the user to select images
5. Creates the destination directory if it doesn't exist
6. Exports each selected image to a .tar file in the destination directory

The exported files follow the naming convention: `<image_name>_<tag>_<os>_<arch>.tar`
- '/' characters in image names are replaced with '·'
- If tag, OS, or architecture info is not available, "latest", "unknown", or "unknown" is used respectively

### Function: DeleteImages
```go
func DeleteImages(grepPattern string)
```

Deletes the selected Docker images.

This function:
1. Initializes a Docker client
2. Lists all Docker images
3. Filters images based on the provided grep pattern
4. Shows a multi-select prompt to the user to select images to delete
5. Deletes each selected image with PruneChildren enabled to remove dependent images too

### Function: CleanCache
```go
func CleanCache()
```

Deletes all files in the cache directory (/tmp/go-dkci).

This function:
1. Checks if the cache directory exists
2. Lists all files in the cache directory
3. Asks for user confirmation before deletion
4. Deletes all files in the directory after confirmation

### Function: ImportImagesFromSource
```go
func ImportImagesFromSource(source string, grepPattern string)
```

Imports Docker images from a specified source file or directory.

Parameters:
- `source`: Path to a .tar file or directory containing .tar files
- `grepPattern`: Pattern to filter files (optional, only used when source is a directory)

If the source is a directory, it searches for .tar, .tar.gz, or .tgz files.
If the source is a file, it imports directly from that file.

## cloud package

### Function: ExportImagesToCloud
```go
func ExportImagesToCloud(cloudPath string)
```

Exports the selected Docker images to Baidu cloud disk.

This function:
1. Gets BDFS configuration using config.GetBDFSConfig()
2. Creates a BDFS client and authorizes it
3. Initializes Docker client
4. Lists all Docker images
5. Filters images based on optional grep pattern (from environment variable DKCI_GREP_PATTERN)
6. Shows a multi-select prompt to the user to select images
7. Exports each selected image to a temporary file in `/tmp/go-dkci`
8. Uploads the temporary file to Baidu cloud at the specified cloudPath
9. Cleans up the temporary file after successful upload

The exported files follow the naming convention: `<image_name>_<tag>_<os>_<arch>.tar`
- '/' characters in image names are replaced with '·'
- If tag, OS, or architecture info is not available, "latest", "unknown", or "unknown" is used respectively

### Function: ImportImagesFromCloud
```go
func ImportImagesFromCloud(cloudPath string, grepPattern string)
```

Downloads Docker images from Baidu cloud disk and imports them to local Docker.

Parameters:
- `cloudPath`: Path to a .tar file or directory in Baidu cloud
- `grepPattern`: Pattern to filter files (optional, only used when cloudPath is a directory)

This function:
1. Gets BDFS configuration using config.GetBDFSConfig()
2. Creates a BDFS client and authorizes it
3. Checks if cloudPath is a file or directory
4. If it's a directory, it lists and filters .tar files based on the grep pattern
5. Shows a multi-select prompt to the user to select files
6. Downloads selected files to temporary location in `/tmp/go-dkci`
7. Imports each downloaded file as a Docker image using docker.ImportImagesFromSource
8. Cleans up temporary files after successful import

The function supports .tar, .tar.gz, and .tgz file formats.
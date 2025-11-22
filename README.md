# go-dkci

A command-line tool for managing Docker images with local storage and Baidu Cloud Disk integration.

## Project Overview

go-dkci is a Go-based tool that helps manage Docker images by providing functionality to:
- Export Docker images to local directories or directly to Baidu Cloud Disk
- Import Docker images from local .tar files or from Baidu Cloud Disk
- Delete Docker images
- Clean temporary cache files

## Features

- **Export**: Export Docker images as .tar files with naming format `<image_name>_<tag>_<os>_<arch>.tar`
- **Import**: Import Docker images from .tar files (including .tar.gz and .tgz)
- **Cloud Integration**: Direct integration with Baidu Cloud Disk for storage
- **Interactive Interface**: User-friendly multi-select interface for choosing images
- **Filtering**: Pattern matching to filter images during operations
- **Clean Operations**: Clean up temporary cache directory

## Installation

### Prerequisites
- Go 1.25.4 or higher
- Docker daemon running locally

### Build from Source

```bash
# Clone the repository
git clone https://github.com/baowuhe/go-dkci.git
cd go-dkci

# Build the binary
go build -o go-dkci main.go

# Or install to your Go bin directory
go install
```

### Using Go Install

```bash
go install github.com/baowuhe/go-dkci@latest
```

## Configuration

### Baidu Cloud Configuration

The tool supports two methods for configuring Baidu Cloud access:

#### 1. Environment Variables
```bash
export BDFS_CLIENT_ID="your_client_id"
export BDFS_CLIENT_SECRET="your_client_secret" 
export BDFS_TOKEN_PATH="/path/to/token/file"
export BDFS_DEFAULT_CLOUD_DIR="/docker-images"  # Optional, defaults to "/"
```

#### 2. Configuration File (TOML format)

Create a configuration file at `~/.local/app/dkci/config.toml`:

```toml
client_id = "your_client_id"
client_secret = "your_client_secret"
token_path = "/path/to/token/file"
default_cloud_dir = "/docker-images"  # Optional, defaults to "/"
```

You can also specify a custom config file path:
```bash
export BDFS_CONFIG_FILE="/path/to/custom/config.toml"
```

## Usage

The tool supports several subcommands:

### Export Images

Export Docker images to local directory or Baidu Cloud:

```bash
# Export to local directory
go-dkci export --destination /tmp/images

# Export to Baidu Cloud
go-dkci export --cloud /docker-images

# Export with pattern filter
go-dkci export --destination /tmp/images --grep alpine

# Export to cloud with pattern filter
go-dkci export --cloud /docker-images --grep nginx
```

### Import Images

Import Docker images from local files or Baidu Cloud:

```bash
# Import from local file
go-dkci import --source /tmp/image.tar

# Import from local directory
go-dkci import --source /tmp/docker-images/

# Import from Baidu Cloud
go-dkci import --cloud /docker-images/my-image.tar

# Import from cloud directory with pattern filter
go-dkci import --cloud /docker-images --grep alpine

# Import from local directory with pattern filter
go-dkci import --source /tmp/docker-images/ --grep alpine
```

### Delete Images

Delete local Docker images:

```bash
# Delete with interactive selection
go-dkci delete

# Delete with pattern filter
go-dkci delete --grep alpine
```

### Clean Cache

Clean the temporary directory (`/tmp/go-dkci`):

```bash
go-dkci clean
```

### Check Version

Display the tool version:

```bash
go-dkci version
```

## File Naming Convention

When exporting images, the tool creates files with the following naming convention:
```
<image_name>_<tag>_<os>_<arch>.tar
```

For example:
- `nginx_latest_linux_amd64.tar`
- `alpine_3.12_linux_arm64.tar`

If the image name contains `/`, it is replaced with `·`:
- `mycompany/myapp` becomes `mycompany·myapp`

## Configuration Priority

Configuration values are loaded in the following priority order:
1. Environment variables (`BDFS_CLIENT_ID`, `BDFS_CLIENT_SECRET`, `BDFS_TOKEN_PATH`)
2. Configuration file specified by `BDFS_CONFIG_FILE` environment variable
3. Default configuration file at `~/.local/app/dkci/config.toml`

## Architecture

The project is organized into the following modules:

- `main.go`: Command-line interface and argument parsing
- `cloud/`: Baidu Cloud Disk integration functionality
- `config/`: Configuration management
- `docker/`: Local Docker operations (export, import, delete)
- `pkg/`: Additional utility packages

## Dependencies

- `github.com/AlecAivazis/survey/v2`: Interactive prompts
- `github.com/baowuhe/go-bdfs`: Baidu Cloud Disk SDK
- `github.com/docker/docker`: Docker API client
- `github.com/pelletier/go-toml/v2`: TOML configuration parsing
- `github.com/spf13/pflag`: Command-line flag parsing

## License

This project is open source and available under the MIT License.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/baowuhe/go-dkci/cloud"
	"github.com/baowuhe/go-dkci/config"
	"github.com/baowuhe/go-dkci/docker"
	"github.com/spf13/pflag"
)

var (
	destination     string
	cloudPath       string
	grepPattern     string
	source          string
	cloudImportPath string
)

// Define the version here - could be set during build time in a real application
var version = "v0.1.0"

func main() {
	// Set up the version command
	versionCmd := pflag.NewFlagSet("version", pflag.ExitOnError)

	// Set up the export command
	exportCmd := pflag.NewFlagSet("export", pflag.ExitOnError)
	exportCmd.StringVarP(&destination, "destination", "d", "/tmp/go-dkci", "Specify the export directory")
	exportCmd.StringVarP(&cloudPath, "cloud", "c", "", "Specify the Baidu cloud folder path for export (mutually exclusive with -d)")
	exportCmd.StringVarP(&grepPattern, "grep", "g", "", "Filter images by pattern")

	// Set up the import command
	importCmd := pflag.NewFlagSet("import", pflag.ExitOnError)
	importCmd.StringVarP(&source, "source", "s", "", "Specify the source .tar file path or directory containing .tar files")
	importCmd.StringVarP(&cloudImportPath, "cloud", "c", "", "Specify the Baidu cloud file or folder path for import (mutually exclusive with -s)")
	importCmd.StringVarP(&grepPattern, "grep", "g", "", "Filter files by pattern")

	// Set up the delete command
	deleteCmd := pflag.NewFlagSet("delete", pflag.ExitOnError)
	deleteCmd.StringVarP(&grepPattern, "grep", "g", "", "Filter images by pattern")

	// Set up the clean command
	cleanCmd := pflag.NewFlagSet("clean", pflag.ExitOnError)

	// Check if there are arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	// Parse the subcommand
	switch os.Args[1] {
	case "export":
		// Check for help flag before full parsing
		showHelp := false
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				showHelp = true
				break
			}
		}

		if showHelp {
			exportCmd.Parse(os.Args[2:])
		} else {
			// Check if flags were explicitly set before parsing
			hasDFlag := false
			hasCFlag := false
			for _, arg := range os.Args[2:] {
				if strings.HasPrefix(arg, "-d") || strings.HasPrefix(arg, "--destination") {
					hasDFlag = true
				}
				if strings.HasPrefix(arg, "-c") || strings.HasPrefix(arg, "--cloud") {
					hasCFlag = true
				}
			}

			exportCmd.Parse(os.Args[2:])

			// Store grep pattern in environment variable for access by other modules
			if grepPattern != "" {
				os.Setenv("DKCI_GREP_PATTERN", grepPattern)
			}

			// Check if both destination and cloud path are specified
			if hasDFlag && cloudPath != "" {
				fmt.Println("[x] Error: -d and -c flags are mutually exclusive")
				os.Exit(1)
			}

			// Check if BDFS configuration is available (to determine if we should use cloud export with default dir)
			bdfsConfigAvailable := false
			if os.Getenv("BDFS_CONFIG_FILE") != "" ||
				(os.Getenv("BDFS_CLIENT_ID") != "" && os.Getenv("BDFS_CLIENT_SECRET") != "" && os.Getenv("BDFS_TOKEN_PATH") != "") {
				bdfsConfigAvailable = true
			}

			if cloudPath != "" {
				cloud.ExportImagesToCloud(cloudPath)
			} else if cloudPath == "" && hasCFlag {
				// If -c flag was explicitly provided with empty value, use default cloud directory from config
				configData, err := config.GetBDFSConfig()
				if err != nil {
					fmt.Printf("[x] Error getting BDFS configuration: %v\n", err)
					os.Exit(1)
				}
				// Use the default cloud directory from config, falling back to "/" if not set
				defaultPath := configData.DefaultCloudDir
				if defaultPath == "" {
					defaultPath = "/"
				}
				cloud.ExportImagesToCloud(defaultPath)
			} else if cloudPath == "" && bdfsConfigAvailable {
				// If cloudPath is empty and BDFS config is provided (but -c not explicitly used), use default cloud directory
				configData, err := config.GetBDFSConfig()
				if err != nil {
					fmt.Printf("[x] Error getting BDFS configuration: %v\n", err)
					os.Exit(1)
				}
				cloud.ExportImagesToCloud(configData.DefaultCloudDir)
			} else {
				docker.ExportImages(destination)
			}
		}
	case "import":
		// Check for help flag before full parsing
		showHelp := false
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				showHelp = true
				break
			}
		}

		if showHelp {
			importCmd.Parse(os.Args[2:])
		} else {
			// Check if flags were explicitly set before parsing
			hasSFlag := false
			hasCFlag := false
			for _, arg := range os.Args[2:] {
				if strings.HasPrefix(arg, "-s") || strings.HasPrefix(arg, "--source") {
					hasSFlag = true
				}
				if strings.HasPrefix(arg, "-c") || strings.HasPrefix(arg, "--cloud") {
					hasCFlag = true
				}
			}

			importCmd.Parse(os.Args[2:])

			// Store grep pattern in environment variable for access by other modules
			if grepPattern != "" {
				os.Setenv("DKCI_GREP_PATTERN", grepPattern)
			}

			// Check if both source and cloud path are specified
			if hasSFlag && cloudImportPath != "" {
				fmt.Println("[x] Error: -s and -c flags are mutually exclusive")
				os.Exit(1)
			}

			if source != "" {
				// Use local source
				docker.ImportImagesFromSource(source, grepPattern)
			} else if cloudImportPath != "" {
				// Use cloud import
				cloud.ImportImagesFromCloud(cloudImportPath, grepPattern)
			} else if cloudImportPath == "" && hasCFlag {
				// If -c flag was explicitly provided with empty value, use default cloud directory from config
				configData, err := config.GetBDFSConfig()
				if err != nil {
					fmt.Printf("[x] Error getting BDFS configuration: %v\n", err)
					os.Exit(1)
				}
				// Use the default cloud directory from config, falling back to "/" if not set
				defaultPath := configData.DefaultCloudDir
				if defaultPath == "" {
					defaultPath = "/"
				}
				cloud.ImportImagesFromCloud(defaultPath, grepPattern)
			} else {
				fmt.Println("[x] Error: either -s/--source or -c/--cloud flag is required for import command")
				os.Exit(1)
			}
		}
	case "delete":
		// Check for help flag before full parsing
		showHelp := false
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				showHelp = true
				break
			}
		}

		if showHelp {
			deleteCmd.Parse(os.Args[2:])
		} else {
			deleteCmd.Parse(os.Args[2:])

			// Store grep pattern in environment variable for access by other modules
			if grepPattern != "" {
				os.Setenv("DKCI_GREP_PATTERN", grepPattern)
			}

			docker.DeleteImages(grepPattern)
		}
	case "version":
		// Check for help flag before full parsing
		showHelp := false
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				showHelp = true
				break
			}
		}

		if showHelp {
			versionCmd.Parse(os.Args[2:])
		} else {
			versionCmd.Parse(os.Args[2:])
			fmt.Printf("go-dkci version %s\n", version)
		}
	case "clean":
		// Check for help flag before full parsing
		showHelp := false
		for _, arg := range os.Args[2:] {
			if arg == "-h" || arg == "--help" {
				showHelp = true
				break
			}
		}

		if showHelp {
			cleanCmd.Parse(os.Args[2:])
		} else {
			cleanCmd.Parse(os.Args[2:])
			docker.CleanCache()
		}
	case "help":
		printUsage()
	case "-h":
		printUsage()
	case "--help":
		printUsage()
	default:
		fmt.Printf("Unrecognized subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("go-dkci - A tool for managing Docker images with Baidu Cloud")
	fmt.Println()
	fmt.Println("Usage: go-dkci [command] [flags]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  export    Export Docker images to local directory or Baidu Cloud")
	fmt.Println("  import    Import Docker images from local .tar files")
	fmt.Println("  delete    Delete Docker images")
	fmt.Println("  clean     Clean cache directory")
	fmt.Println("  version   Print program version")
	fmt.Println("  help      Display this help information")
	fmt.Println()
	fmt.Println("Export command flags:")
	fmt.Println("  -d, --destination string   Specify the export directory (default \"/tmp/go-dkci\")")
	fmt.Println("  -c, --cloud string         Specify the Baidu cloud folder path for export (mutually exclusive with -d)")
	fmt.Println("  -g, --grep string          Filter images by pattern")
	fmt.Println()
	fmt.Println("Import command flags:")
	fmt.Println("  -s, --source string        Specify the source .tar file path or directory containing .tar files")
	fmt.Println("  -c, --cloud string         Specify the Baidu cloud file or folder path for import (mutually exclusive with -s)")
	fmt.Println("  -g, --grep string          Filter files by pattern (optional)")
	fmt.Println()
	fmt.Println("Delete command flags:")
	fmt.Println("  -g, --grep string          Filter images by pattern (optional)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go-dkci export --destination /tmp/images")
	fmt.Println("  go-dkci export --cloud /docker-images")
	fmt.Println("  go-dkci import --source /tmp/image.tar")
	fmt.Println("  go-dkci import --source /tmp/docker-images/ --grep alpine")
	fmt.Println("  go-dkci delete --grep alpine")
	fmt.Println("  go-dkci clean")
	fmt.Println("  go-dkci version")
	fmt.Println("  go-dkci help")
}

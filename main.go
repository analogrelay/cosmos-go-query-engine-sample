package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/spf13/pflag"
)

const (
	// Default values
	defaultEmulatorEndpoint = "https://localhost:8081"
	defaultEmulatorKey      = "C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw=="
	defaultDatabase         = "SampleDB"
	defaultContainer        = "SampleContainer"
)

func main() {
	// Define command line flags
	endpoint := pflag.String("endpoint", defaultEmulatorEndpoint, "Cosmos DB endpoint, defaults to the local emulator")
	key := pflag.String("key", "", "Cosmos DB authentication key (defaults to emulator key for local emulator or Entra ID auth for other endpoints)")
	database := pflag.String("database", defaultDatabase, "Database name to query")
	container := pflag.String("container", defaultContainer, "Container name to query")
	help := pflag.BoolP("help", "h", false, "Show help")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <QUERY>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --endpoint https://myaccount.documents.azure.com:443 \"SELECT * FROM c\"\n", os.Args[0])
	}

	pflag.Parse()

	// Check if help was requested
	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Get the query from the positional arguments
	args := pflag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Error: Query must be provided as a positional argument")
		pflag.Usage()
		os.Exit(1)
	}
	query := args[0]

	// Set the default key if not provided
	if *key == "" {
		// Use the emulator key if using the emulator endpoint
		if *endpoint == defaultEmulatorEndpoint {
			*key = defaultEmulatorKey
		}
	}

	options := &azcosmos.ClientOptions{}
	if *endpoint == defaultEmulatorEndpoint {
		fmt.Fprintf(os.Stderr, "Disabling TLS verification for local emulator endpoint\n")
		// Disable TLS verification for the local emulator
		options.Transport = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	// Create a client
	var client *azcosmos.Client
	var err error

	if *key != "" {
		// Use key-based authentication
		cred, err := azcosmos.NewKeyCredential(*key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create key credential: %v\n", err)
			os.Exit(1)
		}

		client, err = azcosmos.NewClientWithKey(*endpoint, cred, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create Cosmos DB client with key: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Use Entra ID authentication
		fmt.Println("No key provided. Using Entra ID authentication...")
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create Azure Identity credential: %v\n", err)
			os.Exit(1)
		}

		client, err = azcosmos.NewClient(*endpoint, cred, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create Cosmos DB client with Entra ID: %v\n", err)
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Cosmos DB client: %v\n", err)
		os.Exit(1)
	}

	// Get the database and container clients
	containerClient, err := client.NewContainer(*database, *container)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get container client: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Running query: %s\n", query)
	fmt.Printf("Against endpoint: %s\n", *endpoint)
	fmt.Printf("Database: %s, Container: %s\n\n", *database, *container)

	// Execute the query
	pager := containerClient.NewQueryItemsPager(query, azcosmos.NewPartitionKey(), nil)

	// Process the results
	fmt.Println("Results:")
	fmt.Println("---------")

	resultCount := 0
	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch query results: %v\n", err)
			os.Exit(1)
		}

		for _, item := range resp.Items {
			// Unmarshal and remarshal to get a pretty-printed JSON
			var rawJSON map[string]interface{}
			err = json.Unmarshal(item, &rawJSON)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to unmarshal result: %v\n", err)
				continue
			}

			byt, err := json.MarshalIndent(rawJSON, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to marshal result: %v\n", err)
				continue
			}

			// Pretty print the raw JSON
			fmt.Println(string(byt))
			resultCount++
		}
	}

	fmt.Printf("\nQuery complete. Found %d items.\n", resultCount)
}

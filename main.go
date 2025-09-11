package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-cosmos-client-engine/go/azcosmoscx"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/pflag"
)

// generateEmbedding generates a 1536-dimensional vector embedding for the given text using OpenAI or Azure OpenAI
func generateEmbedding(text, apiKey, azureEndpoint, azureDeployment string) ([]float64, error) {
	var client openai.Client
	var model string

	if azureEndpoint != "" {
		// Azure OpenAI configuration
		// Construct the proper Azure OpenAI URL with deployment and API version
		baseURL := strings.TrimSuffix(azureEndpoint, "/") + "/openai/deployments/" + azureDeployment
		client = openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
			option.WithHeader("api-key", apiKey),          // Azure OpenAI uses api-key header
			option.WithQuery("api-version", "2024-02-01"), // Required API version for Azure OpenAI
		)
		model = azureDeployment // Use the deployment name for Azure OpenAI
	} else {
		// Standard OpenAI configuration
		if apiKey != "" {
			os.Setenv("OPENAI_API_KEY", apiKey)
		}
		client = openai.NewClient()
		model = string(openai.EmbeddingModelTextEmbeddingAda002)
	}

	// Create input union from string
	inputUnion := openai.EmbeddingNewParamsInputUnion{
		OfString: openai.String(text),
	}

	// Create the embedding request
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model),
		Input: inputUnion,
	}

	resp, err := client.Embeddings.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

// formatVectorForSQL formats a float64 slice as a SQL array string
func formatVectorForSQL(vector []float64) string {
	strValues := make([]string, len(vector))
	for i, v := range vector {
		strValues[i] = fmt.Sprintf("%.6f", v)
	}
	return "[" + strings.Join(strValues, ",") + "]"
}

// buildVectorSearchQuery constructs a vector similarity search query
func buildVectorSearchQuery(vectorField string, vector []float64, topK int) string {
	vectorStr := formatVectorForSQL(vector)
	return fmt.Sprintf(`SELECT TOP %d c.title, c.text, VectorDistance(c.%s, %s) AS SimilarityScore
FROM c
ORDER BY VectorDistance(c.%s, %s)`, topK, vectorField, vectorStr, vectorField, vectorStr)
}

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
	vectorSearch := pflag.String("vector-search", "", "Text to convert to vector for similarity search")
	vectorField := pflag.String("vector-field", "embedding", "Name of the vector field in your documents")
	topK := pflag.Int("top", 10, "Number of top similar results to return")
	openaiKey := pflag.String("openai-key", "", "OpenAI or Azure OpenAI API key for generating embeddings")
	azureEndpoint := pflag.String("azure-openai-endpoint", "", "Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com/)")
	azureDeployment := pflag.String("azure-openai-deployment", "text-embedding-ada-002", "Azure OpenAI deployment name for embeddings")
	help := pflag.BoolP("help", "h", false, "Show help")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <QUERY>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s [OPTIONS] --vector-search <TEXT>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  Regular query:\n")
		fmt.Fprintf(os.Stderr, "    %s \"SELECT * FROM c\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Vector search with OpenAI:\n")
		fmt.Fprintf(os.Stderr, "    %s --vector-search \"What Bond films have SPECTRE in them?\" --openai-key YOUR_OPENAI_KEY\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Vector search with Azure OpenAI:\n")
		fmt.Fprintf(os.Stderr, "    %s --vector-search \"What Bond films have SPECTRE in them?\" --openai-key YOUR_AZURE_KEY --azure-openai-endpoint https://your-resource.openai.azure.com/ --azure-openai-deployment text-embedding-ada-002\n", os.Args[0])
	}

	pflag.Parse()

	// Check if help was requested
	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Determine if we're doing vector search or regular query
	var query string
	if *vectorSearch != "" {
		// Vector search mode: convert text to vector and build query
		apiKey := *openaiKey
		if apiKey == "" {
			apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
		}
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: OpenAI API key is required for vector search. Use --openai-key or set OPENAI_API_KEY environment variable.")
			os.Exit(1)
		}

		fmt.Printf("Generating embedding for: %s\n", *vectorSearch)
		vector, err := generateEmbedding(*vectorSearch, apiKey, *azureEndpoint, *azureDeployment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate embedding: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generated %d-dimensional vector\n", len(vector))
		query = buildVectorSearchQuery(*vectorField, vector, *topK)
	} else {
		// Regular query mode: get query from positional arguments
		args := pflag.Args()
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Error: Either provide --vector-search flag or query as a positional argument")
			pflag.Usage()
			os.Exit(1)
		}
		query = args[0]
	}

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

	queryOptions := azcosmos.QueryOptions{
		QueryEngine: azcosmoscx.NewQueryEngine(),
	}

	// Execute the query
	pager := containerClient.NewQueryItemsPager(query, azcosmos.NewPartitionKey(), &queryOptions)

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

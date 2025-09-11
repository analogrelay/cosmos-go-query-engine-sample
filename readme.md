# Cosmos Go Query Engine Sample

This sample demonstrates executing Cosmos DB queries using the Go SDK with enhanced query pipeline support in the new azure-cosmos-client-engine.

---

## Prerequisites

- **Go** (version 1.21 or later recommended)
- **Docker** (for running the Cosmos DB Emulator)
- **C Compiler** (required for building the Azure Cosmos Client Engine CGO dependencies)
- **Git** (to clone this repository)

> **Important**: The Azure Cosmos Client Engine requires CGO (C bindings) to be enabled, which means you need a C compiler installed on your system.

---

## Setup Instructions

### 1. Install Required Tools

#### For Ubuntu/Debian

```sh
sudo apt update
sudo apt install golang-go docker.io build-essential git -y
sudo systemctl enable --now docker
```

#### For Windows

**Option 1: Using MSYS2 (Recommended)**

1. **Install MSYS2**:
   - Download from: https://www.msys2.org/
   - Or use Chocolatey: `choco install msys2`

2. **Install MinGW-w64 GCC**:
   ```bash
   # Open MSYS2 terminal and run:
   pacman -S --noconfirm mingw-w64-x86_64-gcc
   ```

3. **Set Environment Variables** (add to your system PATH or run before each build):
   ```powershell
   $env:PATH = "C:\msys64\mingw64\bin;$env:PATH"
   $env:CGO_ENABLED = 1
   ```

**Option 2: Using Visual Studio Build Tools**

1. **Install Visual Studio Build Tools**:
   - Download from: https://visualstudio.microsoft.com/visual-cpp-build-tools/
   - Install with C++ build tools workload

2. **Use Developer Command Prompt** or set up the environment:
   ```cmd
   call "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvars64.bat"
   set CGO_ENABLED=1
   ```

**Option 3: Using WSL2**

If you have WSL2 installed, you can build and run the application in the Linux environment:
```bash
# In WSL2 terminal:
sudo apt install build-essential
export CGO_ENABLED=1
cd /mnt/c/path/to/your/project
go build
```

#### For macOS

```sh
# Install Xcode Command Line Tools
xcode-select --install

# Or install via Homebrew
brew install gcc
```

#### For other systems

You'll need to ensure you have Go, Docker, and Git installed.
In addition, to build the integration with the client engine, you will need a C compiler targeting your deployment platform (like `gcc`, `clang`, or `zig cc`).

### 2.  Clone the Repository

```shell
git clone <repo-url>
cd cosmos-go-query-engine-sample
```

### 3. Run the Cosmos DB Emulator in Docker

```shell
docker run -p 8081:8081 -p 10251-10255:10251-10255 \
  -e AZURE_COSMOS_EMULATOR_PARTITION_COUNT=1 \
  -e AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE=true \
  --name=cosmosdb-emulator \
  mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator
```

Wait until the emulator is fully started (watch the logs for readiness).

### 4.  Trust the Emulator's Certificate (Recommended)

The Cosmos DB Emulator uses a self-signed certificate. To avoid browser and some client warnings, trust the certificate on your host:

Copy the certificate from the running container:

```shell
docker cp cosmosdb-emulator:/tmp/cosmos/appdata/.system/profiles/Client/AppData/Local/CosmosDBEmulator/emulator.pem ./cosmos-emulator-cert.pem
```

If that path does not exist, try:

```shell
docker cp cosmosdb-emulator:/tmp/cosmos/appdata/Packages/DataExplorer/emulator.pem ./cosmos-emulator-cert.pem
```

Add it to your trusted certificates:

```shell
sudo cp ./cosmos-emulator-cert.pem /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

Note: For Go, the sample disables TLS verification for the emulator endpoint, so this step is not strictly required for running the sample, but it is recommended for browser access and other tools.

### 5. Run the Sample

You may need to set `GOPRIVATE` in order to avoid using the public Go module proxy for the (currently private) azure-cosmos-client-engine package:

**For Linux/macOS:**
```shell
export GOPRIVATE=github.com/Azure/azure-cosmos-client-engine
```

**For Windows PowerShell:**
```powershell
$env:GOPRIVATE="github.com/Azure/azure-cosmos-client-engine"
```

**Enable CGO and run the sample:**

**For Linux/macOS:**
```shell
export CGO_ENABLED=1
go run main.go "SELECT * FROM c order by c.number desc OFFSET 1 LIMIT 2"
```

**For Windows (with MSYS2):**
```powershell
$env:PATH = "C:\msys64\mingw64\bin;$env:PATH"
$env:CGO_ENABLED = 1
go run main.go "SELECT * FROM c order by c.number desc OFFSET 1 LIMIT 2"
```

You can also specify options:

```shell
go run main.go --database SampleDB --container SampleContainer "SELECT * FROM c"
```

## Vector Search Support

This sample also demonstrates vector similarity search capabilities using Azure Cosmos DB's vector indexing features combined with OpenAI or Azure OpenAI for generating embeddings.

### Vector Search Prerequisites

1. **OpenAI or Azure OpenAI Account**: You need access to either:
   - OpenAI API with an API key
   - Azure OpenAI Service with a deployed embedding model

2. **Vector-Enabled Container**: Your Cosmos DB container must be configured with vector indexing policy and vector embedding policy for the vector fields.

3. **Document Structure**: Documents in your container must contain vector fields with pre-computed embeddings.

### Required Document Structure

For vector search to work, your documents must have the following structure:

```json
{
  "id": "unique-document-id",
  "title": "Document Title", 
  "text": "Document content for search",
  "embedding": [0.123, -0.456, 0.789, ...], // 1536-dimensional float array
  // ... other fields
}
```

**Key Requirements**:
- **`embedding` field**: Must contain a 1536-dimensional array of floats (for OpenAI text-embedding-ada-002 model)
- **`title` and `text` fields**: Used for displaying search results
- **Vector field name**: Configurable via `--vector-field` parameter (defaults to "embedding")

### Container Configuration

Your Cosmos DB container needs proper vector indexing configuration:

```json
{
  "indexingPolicy": {
    "vectorIndexes": [
      {
        "path": "/embedding",
        "type": "quantizedFlat"
      }
    ]
  },
  "vectorEmbeddingPolicy": {
    "vectorEmbeddings": [
      {
        "path": "/embedding",
        "dataType": "float32",
        "distanceFunction": "cosine",
        "dimensions": 1536
      }
    ]
  }
}
```

### Vector Search Examples

#### Using OpenAI API

```shell
# Set your OpenAI API key
export OPENAI_API_KEY="your-openai-api-key"

# Run vector search
go run main.go --vector-search "What Bond films have SPECTRE in them?" --openai-key "$OPENAI_API_KEY"
```

#### Using Azure OpenAI

**Linux/macOS:**
```shell
# Run vector search with Azure OpenAI
go run main.go \
  --vector-search "What Bond films have SPECTRE in them?" \
  --openai-key "your-azure-openai-key" \
  --azure-openai-endpoint "https://your-resource.openai.azure.com/" \
  --azure-openai-deployment "text-embedding-ada-002"
```

**Windows PowerShell:**
```powershell
# Run vector search with Azure OpenAI
go run main.go --vector-search "What Bond films have SPECTRE in them?" --openai-key "your-azure-openai-key" --azure-openai-endpoint "https://your-resource.openai.azure.com/" --azure-openai-deployment "text-embedding-ada-002"
```

#### Additional Vector Search Options

```shell
# Specify custom vector field name and result count
go run main.go \
  --vector-search "search query text" \
  --vector-field "custom_vector_field_name" \
  --top 5 \
  --openai-key "your-key"

# Use with custom database/container
go run main.go \
  --database "MyDatabase" \
  --container "MyVectorContainer" \
  --vector-search "search query" \
  --openai-key "your-key"
```

### Vector Search Process

1. **Text to Vector**: Your search text is converted to a 1536-dimensional embedding using OpenAI/Azure OpenAI
2. **Similarity Query**: The generated vector is compared against document embeddings using cosine distance
3. **Ranked Results**: Documents are returned ranked by similarity score (higher scores = more similar)

### Building Vector-Enabled Applications

To create your own vector search application:

1. **Prepare Documents**: Ensure your documents have vector embeddings pre-computed and stored
2. **Configure Container**: Set up vector indexing and embedding policies
3. **Generate Embeddings**: Use the same embedding model for both document storage and search queries
4. **Query Structure**: The app generates SQL queries like:
   ```sql
   SELECT TOP 10 c.title, c.text, VectorDistance(c.embedding, [0.1, -0.2, ...]) AS SimilarityScore
   FROM c
   ORDER BY VectorDistance(c.embedding, [0.1, -0.2, ...])
   ```

## Command Line Options

The application supports both regular SQL queries and vector similarity search:

### General Options
- `--endpoint`: Cosmos DB endpoint (default: local emulator)
- `--key`: Authentication key (uses emulator key for local, or Entra ID for Azure)
- `--database`: Database name (default: "fabcon25demo")
- `--container`: Container name (default: "search_diskann")
- `--help`, `-h`: Show help information

### Vector Search Options
- `--vector-search`: Text to convert to vector for similarity search
- `--vector-field`: Name of vector field in documents (default: "embedding")
- `--top`: Number of results to return (default: 10)
- `--openai-key`: OpenAI or Azure OpenAI API key
- `--azure-openai-endpoint`: Azure OpenAI endpoint URL
- `--azure-openai-deployment`: Azure OpenAI deployment name (default: "text-embedding-ada-002")

### Usage Examples

**All Platforms:**
```shell
# Regular SQL query
go run main.go "SELECT * FROM c WHERE c.title LIKE '%Bond%'"

# Vector search with OpenAI
go run main.go --vector-search "action movies" --openai-key "your-key"

# Show help
go run main.go --help
```

**Linux/macOS (multi-line commands):**
```shell
# Vector search with Azure OpenAI  
go run main.go --vector-search "action movies" \
  --openai-key "your-azure-key" \
  --azure-openai-endpoint "https://your-resource.openai.azure.com/" \
  --azure-openai-deployment "embeddings"

# Custom database/container
go run main.go --database "Movies" --container "Films" \
  --vector-search "romantic comedies" --openai-key "your-key"
```

**Windows PowerShell:**
```powershell
# Vector search with Azure OpenAI (single line)
go run main.go --vector-search "action movies" --openai-key "your-azure-key" --azure-openai-endpoint "https://your-resource.openai.azure.com/" --azure-openai-deployment "embeddings"

# Custom database/container
go run main.go --database "Movies" --container "Films" --vector-search "romantic comedies" --openai-key "your-key"
```

## Performance Optimization

### For Faster Execution

**Build Once, Run Multiple Times:**
```powershell
# Build the executable once
go build -o cosmos-search.exe

# Then run multiple times without recompilation
.\cosmos-search.exe --vector-search "search query" --openai-key "your-key" --azure-openai-endpoint "https://your-endpoint/" --azure-openai-deployment "embeddings"
```

**Performance Factors:**
- **Compilation Time**: `go run` recompiles every time; `go build` compiles once
- **Azure OpenAI Latency**: Network calls to Azure OpenAI for embedding generation (~1-3 seconds)
- **Vector Search**: Cosmos DB computes similarity against all documents in container
- **CGO Overhead**: C bindings add compilation and runtime overhead

### Typical Execution Times
- **First run with `go run`**: 10-15 seconds (includes compilation)
- **Subsequent runs with `go run`**: 10-15 seconds (recompiles each time)
- **Using pre-built executable**: 3-5 seconds (no compilation)
- **Azure OpenAI API call**: 1-3 seconds (network latency)

## Notes

- **No Build Required**: Use `go run main.go` to compile and run directly - no need to create executable files
- **C Compiler Required**: The Azure Cosmos Client Engine requires CGO (C bindings), which means a C compiler is mandatory for building this application.
- **CGO Must Be Enabled**: Ensure `CGO_ENABLED=1` is set in your environment when building or running the application.
- **Windows PowerShell**: Use single-line commands (no line continuation with `\` like in bash)
- **Emulator Limitations**: The Cosmos DB Emulator is officially supported on Windows. The Linux Docker version is in preview and may have limitations.

## Troubleshooting

### General Issues

- If you see connection errors, ensure the emulator is running and accessible on <https://localhost:8081>.
- If you see "Owner resource does not exist", ensure the database and container exist.
- For certificate issues, see [the official repo.](https://github.com/Azure/azure-cosmos-db-emulator-docker)

### Windows-Specific CGO Issues

**Problem**: `build constraints exclude all Go files` when building
**Solution**: This means CGO is disabled or no C compiler is found. Follow the Windows setup instructions above.

**Problem**: `gcc: command not found` or similar C compiler errors
**Solution**: 
1. Install MSYS2 and MinGW-w64 as described above
2. Add `C:\msys64\mingw64\bin` to your PATH
3. Ensure `CGO_ENABLED=1` is set

**Problem**: `cgo: C compiler "gcc" not found`
**Solution**: 
```powershell
# Add MSYS2 to PATH and enable CGO
$env:PATH = "C:\msys64\mingw64\bin;$env:PATH"
$env:CGO_ENABLED = 1

# Verify gcc is available
gcc --version

# Then build/run your application
go run main.go "SELECT * FROM c"
```

**Problem**: Build works but runtime errors about missing DLLs
**Solution**: Ensure the MinGW-w64 bin directory is in your PATH when running the application, not just when building.

### Quick Setup Script for Windows

Create a batch file `setup-env.bat` for easy setup:

```batch
@echo off
echo Setting up environment for Azure Cosmos Client Engine...
set PATH=C:\msys64\mingw64\bin;%PATH%
set CGO_ENABLED=1
echo Environment ready! You can now run: go run main.go "SELECT * FROM c"
cmd /k
```

Run this script before building or running your Go application.

# Cosmos Go Query Engine Sample

This sample demonstrates executing Cosmos DB queries using the Go SDK with enhanced query pipeline support in the new azure-cosmos-client-engine.

---

## Prerequisites

- **Go** (version 1.21 or later recommended)
- **Docker** (for running the Cosmos DB Emulator)
- **C Compiler** (required for building some Go dependencies; install with `sudo apt install build-essential`)
- **Git** (to clone this repository)

---

## Setup Instructions

### 1. Install Required Tools

```sh
sudo apt update
sudo apt install golang-go docker.io build-essential git -y
sudo systemctl enable --now docker
```

### 2.  Clone the Repository

```shell
git clone <repo-url>
cd cosmos-go-query-engine-sample
```

### 3. Run the Cosmos DB Emulator in Docker

```shell
sudo docker pull mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator
sudo docker run -p 8081:8081 -p 10251-10255:10251-10255 \
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
sudo docker cp cosmosdb-emulator:/tmp/cosmos/appdata/.system/profiles/Client/AppData/Local/CosmosDBEmulator/emulator.pem ./cosmos-emulator-cert.pem
```

If that path does not exist, try:

```shell
sudo docker cp cosmosdb-emulator:/tmp/cosmos/appdata/Packages/DataExplorer/emulator.pem ./cosmos-emulator-cert.pem
```

Add it to your trusted certificates:

```shell
sudo cp ./cosmos-emulator-cert.pem /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

Note: For Go, the sample disables TLS verification for the emulator endpoint, so this step is not strictly required for running the sample, but it is recommended for browser access and other tools.

### 5. Run the Sample

Run a query (note: you would need to have created the database and container with appropriate data):

```shell
go run main.go "SELECT * FROM c order by c.number desc OFFSET 1 LIMIT 2"
```

You can also specify options:

```shell
go run main.go --database SampleDB --container SampleContainer "SELECT * FROM c"
```

## Notes
- C Compiler Required: Some Go dependencies require a C compiler. Make sure you have build-essential installed.
- Emulator Limitations: The Cosmos DB Emulator is officially supported on Windows. The Linux Docker version is in preview and may have limitations.
- Database/Container: The sample will attempt to create the database and container if they do not exist.

## Troubleshooting
- If you see connection errors, ensure the emulator is running and accessible on https://localhost:8081.
- If you see "Owner resource does not exist", ensure the database and container exist (the sample will create them if missing).
- For certificate issues, see [the official repo.](https://github.com/Azure/azure-cosmos-db-emulator-docker)
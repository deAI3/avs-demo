# Runnign as an Operator for AVS

## Requiments

- Git
- Golang : version > 1.22
- Already registered as an Operator in Restaking core

## Minimum hardware requirements:

| Component     | Specification     |
|---------------|-------------------|
| **CPU**       | 16 cores          |
| **Memory**    | 32 GB RAM         |
| **Bandwidth** | 1 Gbps            |
| **Storage**   | 256 GB disk space |

## Steps to running an Operator

Step 1: Clone the repository

```bash
git clone https://github.com/A2Dsquad/avs
cd avs
```

Step 2: Build the Operator binary

```bash
go build -mod=readonly -o ./build/avs ./cmd
```

Step 3: Configure the Operator

```bash
./build/avs operator config {avs address} {Aggregator rpc}
```

This will create a `operator-config.json` file in `config` folder. You can see the example in `config/example.json`.

Step 4: Start Operator
```bash
./build/avs operator start
```

Your operator is now up and running!
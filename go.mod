module avs

require (
	github.com/Layr-Labs/eigensdk-go v0.1.12
	github.com/aptos-labs/aptos-go-sdk v0.7.0
	github.com/ethereum/go-ethereum v1.14.5
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.8.1
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.27.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/bits-and-blooms/bitset v1.10.0 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/cosmos/crypto v0.1.2 // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gin-gonic/gin v1.10.1
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3
	github.com/hasura/go-graphql-client v0.12.1 // indirect
	github.com/hdevalence/ed25519consensus v0.2.0 // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/supranational/blst v0.3.13 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	nhooyr.io/websocket v1.8.11 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace (
	github.com/aptos-labs/aptos-go-sdk => github.com/decentrio/aptos-go-sdk v0.0.0-20241007104016-43369f521d0e
	github.com/cosmos/crypto => github.com/decentrio/crypto v0.1.3-0.20240927062649-7a497320b85c
	github.com/decentrio/oracle-avs => ./..
)

go 1.22.6

module aiagent

go 1.23.2

require (
	go.mongodb.org/mongo-driver v1.17.2 // MongoDB driver for data storage
	go.uber.org/zap v1.27.0 // Structured logging
)

require (
	github.com/golang/snappy v0.0.4 // indirect; Dependency for MongoDB driver
	github.com/klauspost/compress v1.16.7 // indirect; Dependency for MongoDB driver
	github.com/montanaflynn/stats v0.7.1 // indirect; Dependency for MongoDB driver
	github.com/stretchr/testify v1.10.0 // indirect; Testing dependency
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect; Dependency for MongoDB driver
	github.com/xdg-go/scram v1.1.2 // indirect; Dependency for MongoDB driver
	github.com/xdg-go/stringprep v1.0.4 // indirect; Dependency for MongoDB driver
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect; Dependency for MongoDB driver
	go.uber.org/multierr v1.10.0 // indirect; Dependency for Zap
	golang.org/x/crypto v0.26.0 // indirect; Dependency for MongoDB driver and crypto/aes
	golang.org/x/sync v0.8.0 // indirect; Dependency for MongoDB driver
	golang.org/x/text v0.17.0 // indirect; Dependency for MongoDB driver
)

// Notes:
// - Versions are specified for consistency, but running `go mod tidy` will resolve the latest compatible versions.
// - Indirect dependencies are included as they will be populated by `go mod tidy` based on the direct dependencies.

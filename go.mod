module aiagent

go 1.23.2

require (
	github.com/gorilla/websocket v1.5.3 // Added for WebSocket support
	github.com/spf13/viper v1.19.0
	go.mongodb.org/mongo-driver v1.17.2
	go.uber.org/zap v1.27.0
)

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect; Dependency for Viper
	github.com/golang/snappy v0.0.4 // indirect; Dependency for MongoDB driver
	github.com/hashicorp/hcl v1.0.0 // indirect; Dependency for Viper
	github.com/klauspost/compress v1.17.2 // indirect; Dependency for MongoDB driver
	github.com/magiconair/properties v1.8.7 // indirect; Dependency for Viper
	github.com/mitchellh/mapstructure v1.5.0 // indirect; Dependency for Viper
	github.com/montanaflynn/stats v0.7.1 // indirect; Dependency for MongoDB driver
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect; Dependency for Viper
	github.com/sagikazarmark/locafero v0.6.0 // indirect; Dependency for Viper
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect; Dependency for Viper
	github.com/sourcegraph/conc v0.3.0 // indirect; Dependency for Viper
	github.com/spf13/afero v1.11.0 // indirect; Dependency for Viper
	github.com/spf13/cast v1.7.0 // indirect; Dependency for Viper
	github.com/spf13/pflag v1.0.5 // indirect; Dependency for Viper
	github.com/stretchr/testify v1.10.0 // indirect; Testing dependency
	github.com/subosito/gotenv v1.6.0 // indirect; Dependency for Viper
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect; Dependency for MongoDB driver
	github.com/xdg-go/scram v1.1.2 // indirect; Dependency for MongoDB driver
	github.com/xdg-go/stringprep v1.0.4 // indirect; Dependency for MongoDB driver
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect; Dependency for MongoDB driver
	go.uber.org/multierr v1.10.0 // indirect; Dependency for Zap
	golang.org/x/crypto v0.26.0 // indirect; Dependency for MongoDB driver and crypto/aes
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect; Dependency for Viper
	golang.org/x/sync v0.8.0 // indirect; Dependency for MongoDB driver and Viper
	golang.org/x/sys v0.25.0 // indirect; Dependency for Viper
	golang.org/x/text v0.17.0 // indirect; Dependency for MongoDB driver and Viper
	gopkg.in/ini.v1 v1.67.0 // indirect; Dependency for Viper
	gopkg.in/yaml.v3 v3.0.1 // indirect; Dependency for Viper
)

require github.com/google/uuid v1.6.0

require github.com/aymerick/raymond v2.0.2+incompatible // indirect

// Notes:
// - Versions are specified for consistency; `go mod tidy` will resolve the latest compatible versions.
// - Indirect dependencies are included as they will be populated by `go mod tidy` based on the direct dependencies.

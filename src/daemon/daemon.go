package daemon

import (
	"log"
	"util/serializer"
)

// AllSparkConfig - allspark configuration parameters struct
type AllSparkConfig struct {
	RedisHost             string
	RedisPassword         string
	ClusterPendingTimeout int64
	ClusterIdleTimeout    int64
	ClusterMaxRuntime     int64
	CloudEnvironment      string
	CallbackURL           string
}

var config AllSparkConfig

// Init - loads allspark configuration parameters into configParams
func Init(path string) {
	err := serializer.DeserializePath(path, &config)
	if err != nil {
		log.Fatal(err)
	}
}

// GetAllSparkConfig - returns allspark configuration parameters
func GetAllSparkConfig() AllSparkConfig {
	return config
}

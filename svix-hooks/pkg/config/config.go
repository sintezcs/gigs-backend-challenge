package config

import (
    "github.com/caarlos0/env/v9"
)

// Config struct
type Config struct {
    Port            int    `env:"PORT" envDefault:"8080"`
    NumWorkers      int32  `env:"NUM_WORKERS" envDefault:"5"`
    ChannelBuffer   int    `env:"CHANNEL_BUFFER" envDefault:"100"`
    SvixApiKey      string `env:"SVIX_API_KEY,required"`
    SvixAppId       string `env:"SVIX_APP_ID,required"`
    SvixApiMaxRate  int    `env:"SVIX_API_MAX_RATE" envDefault:"5"`   // 5 requests per second
    SvixApiMaxBurst int    `env:"SVIX_API_MAX_BURST" envDefault:"10"` // 5 requests per second
}

// ParserOptions options for parsing the config struct
var ParserOptions = env.Options{UseFieldNameByDefault: true}

package limiter

type GlobalDeviceLimitConfig struct {
	Enable        bool   `mapstructure:"Enable"`
	RedisAddr     string `mapstructure:"RedisAddr"` // host:port
	RedisPassword string `mapstructure:"RedisPassword"`
	RedisDB       int    `mapstructure:"RedisDB"`
	Expiry        int    `mapstructure:"Expiry"` // minute
}

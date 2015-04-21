package configVars

type Configuration struct {
	BaseUrl string `envconfig:"BASE_URL"`
	Port    string `envconfig:"PORT"`

	DbUrl                   string `envconfig:"DB_URL"`
	MaxConcurrentDbRequests int    `envconfig:"MAX_CONC_REQS"`

	FbAppId     string `envconfig:"FB_APP_ID"`
	FbAppSecret string `envconfig:"FB_APP_SECRET"`
}

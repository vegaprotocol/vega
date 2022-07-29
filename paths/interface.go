package paths

type Paths interface {
	CreateCachePathFor(CachePath) (string, error)
	CreateCacheDirFor(CachePath) (string, error)
	CreateConfigPathFor(ConfigPath) (string, error)
	CreateConfigDirFor(ConfigPath) (string, error)
	CreateDataPathFor(DataPath) (string, error)
	CreateDataDirFor(DataPath) (string, error)
	CreateStatePathFor(StatePath) (string, error)
	CreateStateDirFor(StatePath) (string, error)
	CachePathFor(CachePath) string
	ConfigPathFor(ConfigPath) string
	DataPathFor(DataPath) string
	StatePathFor(StatePath) string
}

// New instantiates the specific implementation of the Paths interface based on
// the value of the customHome. If a customHome is specified the custom
// implementation CustomPaths is returned, the standard DefaultPaths otherwise.
func New(customHome string) Paths {
	if len(customHome) != 0 {
		return &CustomPaths{
			CustomHome: customHome,
		}
	}

	return &DefaultPaths{}
}

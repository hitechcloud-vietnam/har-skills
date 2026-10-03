package har

// Option defines a parser option.
type Option func(*options)

// options is the internal structure for parser configuration.
type options struct {
	// Whether to enable lenient parsing and attempt to parse valid portions.
	lenient bool
	// Whether to skip validation.
	skipValidation bool
	// Whether to collect all parsing warnings without failing.
	collectWarnings bool
	// Maximum number of warnings allowed before parsing stops.
	maxWarnings int
	// Whether to use memory-optimized structures.
	useMemoryOptimized bool
	// Whether to use lazy loading.
	useLazyLoading bool
	// Whether to use streaming.
	useStreaming bool
	// HAR version to use.
	harVersion string
	// Whether to detect the version automatically.
	autoDetectVersion bool
}

// Default options.
var defaultOptions = options{
	lenient:            false,
	skipValidation:     false,
	collectWarnings:    false,
	maxWarnings:        100,
	useMemoryOptimized: false,
	useLazyLoading:     false,
	useStreaming:       false,
	harVersion:         HarSpecVersion12, // Default version: 1.2.
	autoDetectVersion:  true,             // Enabled by default.
}

// Convert options to the legacy ParseOptions structure.
func (o *options) toParseOptions() ParseOptions {
	return ParseOptions{
		Lenient:         o.lenient,
		SkipValidation:  o.skipValidation,
		CollectWarnings: o.collectWarnings,
		MaxWarnings:     o.maxWarnings,
	}
}

// WithLenient enables lenient parsing.
func WithLenient() Option {
	return func(o *options) {
		o.lenient = true
	}
}

// WithSkipValidation skips validation.
func WithSkipValidation() Option {
	return func(o *options) {
		o.skipValidation = true
	}
}

// WithCollectWarnings collects warnings instead of failing.
func WithCollectWarnings() Option {
	return func(o *options) {
		o.collectWarnings = true
	}
}

// WithMaxWarnings sets the maximum number of warnings.
func WithMaxWarnings(max int) Option {
	return func(o *options) {
		o.maxWarnings = max
	}
}

// WithMemoryOptimized uses memory-optimized structures.
func WithMemoryOptimized() Option {
	return func(o *options) {
		o.useMemoryOptimized = true
	}
}

// WithLazyLoading enables lazy loading.
func WithLazyLoading() Option {
	return func(o *options) {
		o.useLazyLoading = true
	}
}

// WithStreaming enables streaming.
func WithStreaming() Option {
	return func(o *options) {
		o.useStreaming = true
	}
}

// WithHarVersion specifies the HAR version.
func WithHarVersion(version string) Option {
	return func(o *options) {
		if IsValidHarVersion(version) {
			o.harVersion = version
			o.autoDetectVersion = false
		}
	}
}

// WithAutoDetectVersion enables automatic HAR version detection.
func WithAutoDetectVersion(enabled bool) Option {
	return func(o *options) {
		o.autoDetectVersion = enabled
	}
}

// applyOptions applies options to the defaults and returns the result.
func applyOptions(opts ...Option) options {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// Define common option combinations.
var (
	// OptMemoryEfficient is a memory-efficient configuration.
	OptMemoryEfficient = []Option{
		WithMemoryOptimized(),
		WithSkipValidation(),
	}

	// OptFast configures fast parsing.
	OptFast = []Option{
		WithSkipValidation(),
	}

	// OptLenient configures lenient parsing.
	OptLenient = []Option{
		WithLenient(),
		WithCollectWarnings(),
	}

	// OptPerformance is a high-performance configuration.
	OptPerformance = []Option{
		WithMemoryOptimized(),
		WithSkipValidation(),
		WithLazyLoading(),
	}
)

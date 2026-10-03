package har

import (
	"net/http"
	"reflect"
	"time"
)

// FilterOption defines a functional option for filtering.
type FilterOption func(*FilterOptions)

// WithFilterURL sets the URL filter.
func WithFilterURL(url string) FilterOption {
	return func(o *FilterOptions) {
		o.URL = url
	}
}

// WithFilterMethod sets the request method filter.
func WithFilterMethod(method string) FilterOption {
	return func(o *FilterOptions) {
		o.Method = method
	}
}

// WithFilterStatusCode sets the status-code filter.
func WithFilterStatusCode(code int) FilterOption {
	return func(o *FilterOptions) {
		o.StatusCode = code
	}
}

// WithFilterStatusCodeRange sets the status-code range filter.
func WithFilterStatusCodeRange(min, max int) FilterOption {
	return func(o *FilterOptions) {
		o.StatusCodeMin = min
		o.StatusCodeMax = max
	}
}

// WithFilterContentType sets the content-type filter.
func WithFilterContentType(contentType string) FilterOption {
	return func(o *FilterOptions) {
		o.ContentType = contentType
	}
}

// WithFilterTimeRange sets the time-range filter.
func WithFilterTimeRange(start, end time.Time) FilterOption {
	return func(o *FilterOptions) {
		o.StartTime = start
		o.EndTime = end
	}
}

// WithFilterDuration sets the duration filter.
func WithFilterDuration(min, max float64) FilterOption {
	return func(o *FilterOptions) {
		o.MinDuration = min
		o.MaxDuration = max
	}
}

// WithFilterResourceType sets the resource-type filter.
func WithFilterResourceType(resourceType string) FilterOption {
	return func(o *FilterOptions) {
		o.ResourceType = resourceType
	}
}

// WithFilterHasError filters for requests with errors only.
func WithFilterHasError() FilterOption {
	return func(o *FilterOptions) {
		o.HasError = true
	}
}

// WithFilterHeader sets the request-header filter.
func WithFilterHeader(name, value string) FilterOption {
	return func(o *FilterOptions) {
		o.HeaderName = name
		o.HeaderValue = value
	}
}

// WithFilterResponseHeader sets the response-header filter.
func WithFilterResponseHeader(name, value string) FilterOption {
	return func(o *FilterOptions) {
		o.RespHeaderName = name
		o.RespHeaderValue = value
	}
}

// WithFilterRegex enables regular-expression matching.
func WithFilterRegex() FilterOption {
	return func(o *FilterOptions) {
		o.UseRegex = true
	}
}

// NewFilterOptions creates filter options from functional options.
func NewFilterOptions(opts ...FilterOption) FilterOptions {
	options := FilterOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

// FilterWith filters HAR entries using functional options.
func (h *Har) FilterWith(opts ...FilterOption) *FilterResult {
	return h.Filter(NewFilterOptions(opts...))
}

// ReplayOption defines a functional option for replay settings.
type ReplayOption func(*ReplayOptions)

// WithReplayTimeout sets the request timeout.
func WithReplayTimeout(timeout time.Duration) ReplayOption {
	return func(o *ReplayOptions) {
		o.Timeout = timeout
	}
}

// WithReplayFollowRedirects sets whether to follow redirects.
func WithReplayFollowRedirects(follow bool) ReplayOption {
	return func(o *ReplayOptions) {
		o.FollowRedirects = follow
	}
}

// WithReplayMaxRedirects sets the maximum number of redirects.
func WithReplayMaxRedirects(max int) ReplayOption {
	return func(o *ReplayOptions) {
		o.MaxRedirects = max
	}
}

// WithReplaySkipSSLVerify sets whether to skip SSL certificate verification.
func WithReplaySkipSSLVerify(skip bool) ReplayOption {
	return func(o *ReplayOptions) {
		o.SkipSSLVerify = skip
	}
}

// WithReplayOverrideHeader sets request headers to override.
func WithReplayOverrideHeader(name, value string) ReplayOption {
	return func(o *ReplayOptions) {
		if o.OverrideHeaders == nil {
			o.OverrideHeaders = make(map[string]string)
		}
		o.OverrideHeaders[name] = value
	}
}

// WithReplayTransport sets a custom Transport.
func WithReplayTransport(transport interface{}) ReplayOption {
	return func(o *ReplayOptions) {
		if t, ok := transport.(http.RoundTripper); ok && !isNilReplayTransport(t) {
			o.Transport = t
		}
	}
}

func isNilReplayTransport(transport http.RoundTripper) bool {
	if transport == nil {
		return true
	}

	value := reflect.ValueOf(transport)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// NewReplayOptions creates replay options from functional options.
func NewReplayOptions(opts ...ReplayOption) ReplayOptions {
	options := DefaultReplayOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

// ReplayAllWith replays all requests using functional options.
func (h *Har) ReplayAllWith(opts ...ReplayOption) ([]*ReplayResult, error) {
	return h.ReplayAll(NewReplayOptions(opts...))
}

// ConvertOption defines a functional option for conversion settings.
type ConvertOption func(*ConvertOptions)

// WithConvertIncludeHeaders sets whether to include headers.
func WithConvertIncludeHeaders(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeHeaders = include
	}
}

// WithConvertIncludeTimings sets whether to include timings.
func WithConvertIncludeTimings(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeTimings = include
	}
}

// WithConvertIncludeBodies sets whether to include request bodies.
func WithConvertIncludeBodies(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludePostData = include
	}
}

// WithConvertIncludeCookies sets whether to include cookies.
func WithConvertIncludeCookies(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		// Cookies are included via headers; no separate field in ConvertOptions
	}
}

// WithConvertIncludeQueryStrings sets whether to include query strings.
func WithConvertIncludeQueryStrings(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeQueryString = include
	}
}

// WithConvertIncludeStatus sets whether to include status codes.
func WithConvertIncludeStatus(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeStatus = include
	}
}

// WithConvertIncludeSize sets whether to include sizes.
func WithConvertIncludeSize(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeSize = include
	}
}

// WithConvertIncludeURL sets whether to include URLs.
func WithConvertIncludeURL(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeURL = include
	}
}

// WithConvertIncludeMethod sets whether to include the method.
func WithConvertIncludeMethod(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeMethod = include
	}
}

// WithConvertIncludeTime sets whether to include timings.
func WithConvertIncludeTime(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeTime = include
	}
}

// WithConvertIncludeMimeType sets whether to include MIME types.
func WithConvertIncludeMimeType(include bool) ConvertOption {
	return func(o *ConvertOptions) {
		o.IncludeContentType = include
	}
}

// WithConvertHeaders sets a custom header list.
func WithConvertHeaders(headers []string) ConvertOption {
	return func(o *ConvertOptions) {
		o.Headers = headers
	}
}

// WithConvertFilter sets filtering options.
func WithConvertFilter(filter FilterOptions) ConvertOption {
	return func(o *ConvertOptions) {
		o.Filter = &filter
	}
}

// NewConvertOptions creates conversion options from functional options.
func NewConvertOptions(opts ...ConvertOption) ConvertOptions {
	options := DefaultConvertOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

// ConvertWith converts HAR data using functional options.
func (h *Har) ConvertWith(format ConvertFormat, opts ...ConvertOption) (string, error) {
	return h.Convert(format, NewConvertOptions(opts...))
}

// DiffOption defines a functional option for diff settings.
type DiffOption func(*DiffOptions)

// WithDiffIgnoreHeaders sets headers to ignore.
func WithDiffIgnoreHeaders(headers ...string) DiffOption {
	return func(o *DiffOptions) {
		o.IgnoreHeaders = append(o.IgnoreHeaders, headers...)
	}
}

// WithDiffIgnoreTimings sets whether to ignore timing differences.
func WithDiffIgnoreTimings(ignore bool) DiffOption {
	return func(o *DiffOptions) {
		o.IgnoreTimings = ignore
	}
}

// WithDiffIgnoreDates sets whether to ignore date differences.
func WithDiffIgnoreDates(ignore bool) DiffOption {
	return func(o *DiffOptions) {
		o.IgnoreDates = ignore
	}
}

// WithDiffIgnoreCache sets whether to ignore cache differences.
func WithDiffIgnoreCache(ignore bool) DiffOption {
	return func(o *DiffOptions) {
		o.IgnoreCache = ignore
	}
}

// WithDiffIgnoreComment sets whether to ignore comment differences.
func WithDiffIgnoreComment(ignore bool) DiffOption {
	return func(o *DiffOptions) {
		o.IgnoreComment = ignore
	}
}

// WithDiffNormalizeURL sets whether to normalize URLs.
func WithDiffNormalizeURL(normalize bool) DiffOption {
	return func(o *DiffOptions) {
		o.NormalizeURL = normalize
	}
}

// WithDiffCompareByURL sets whether to match entries by URL.
func WithDiffCompareByURL(compare bool) DiffOption {
	return func(o *DiffOptions) {
		o.CompareByURL = compare
	}
}

// WithDiffIncludeBody sets whether to compare response bodies.
func WithDiffIncludeBody(include bool) DiffOption {
	return func(o *DiffOptions) {
		o.IncludeBody = include
	}
}

// NewDiffOptions creates diff options from functional options.
func NewDiffOptions(opts ...DiffOption) DiffOptions {
	options := DefaultDiffOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

// DiffWith compares two HAR files using functional options.
func DiffWith(har1, har2 *Har, opts ...DiffOption) *HarDiff {
	return Diff(har1, har2, NewDiffOptions(opts...))
}

// MergeOption defines a functional option for merge settings.
type MergeOption func(*MergeOptions)

// WithMergeSortByTime sets whether to sort by time.
func WithMergeSortByTime(sort bool) MergeOption {
	return func(o *MergeOptions) {
		o.SortByTime = sort
	}
}

// WithMergeDeduplicate sets whether to deduplicate.
func WithMergeDeduplicate(dedup bool) MergeOption {
	return func(o *MergeOptions) {
		o.Deduplicate = dedup
	}
}

// NewMergeOptions creates merge options from functional options.
func NewMergeOptions(opts ...MergeOption) MergeOptions {
	options := DefaultMergeOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

// MergeWith merges multiple HAR files using functional options.
func MergeWith(opts ...MergeOption) func(hars ...*Har) *Har {
	options := NewMergeOptions(opts...)
	return func(hars ...*Har) *Har {
		return MergeWithOptions(options, hars...)
	}
}

// HarBuilderOption defines a functional option for HarBuilder.
type HarBuilderOption func(*HarBuilder)

// WithBuilderVersion sets the HAR version.
func WithBuilderVersion(version string) HarBuilderOption {
	return func(b *HarBuilder) {
		b.SetVersion(version)
	}
}

// WithBuilderCreator sets creator information.
func WithBuilderCreator(name, version string) HarBuilderOption {
	return func(b *HarBuilder) {
		b.SetCreator(name, version)
	}
}

// WithBuilderBrowser sets browser information.
func WithBuilderBrowser(name, version string) HarBuilderOption {
	return func(b *HarBuilder) {
		b.SetBrowser(name, version)
	}
}

// WithBuilderComment sets the comment.
func WithBuilderComment(comment string) HarBuilderOption {
	return func(b *HarBuilder) {
		b.SetComment(comment)
	}
}

// NewHarBuilderWithOptions creates a HAR builder using functional options.
func NewHarBuilderWithOptions(opts ...HarBuilderOption) *HarBuilder {
	builder := NewHarBuilder()
	for _, opt := range opts {
		if opt != nil {
			opt(builder)
		}
	}
	return builder
}

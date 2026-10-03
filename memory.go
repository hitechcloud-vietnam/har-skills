package har

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// HTTPMethod is an enum of HTTP methods that reduces string memory usage.
type HTTPMethod uint8

const (
	MethodUnknown HTTPMethod = iota
	MethodGET
	MethodPOST
	MethodPUT
	MethodDELETE
	MethodHEAD
	MethodOPTIONS
	MethodPATCH
	MethodCONNECT
	MethodTRACE
)

// methodToString converts an HTTPMethod to a string.
var methodToString = map[HTTPMethod]string{
	MethodUnknown: "UNKNOWN",
	MethodGET:     "GET",
	MethodPOST:    "POST",
	MethodPUT:     "PUT",
	MethodDELETE:  "DELETE",
	MethodHEAD:    "HEAD",
	MethodOPTIONS: "OPTIONS",
	MethodPATCH:   "PATCH",
	MethodCONNECT: "CONNECT",
	MethodTRACE:   "TRACE",
}

// stringToMethod converts a string to an HTTPMethod.
var stringToMethod = map[string]HTTPMethod{
	"GET":     MethodGET,
	"POST":    MethodPOST,
	"PUT":     MethodPUT,
	"DELETE":  MethodDELETE,
	"HEAD":    MethodHEAD,
	"OPTIONS": MethodOPTIONS,
	"PATCH":   MethodPATCH,
	"CONNECT": MethodCONNECT,
	"TRACE":   MethodTRACE,
}

// String returns the string representation of an HTTPMethod.
func (m HTTPMethod) String() string {
	if s, ok := methodToString[m]; ok {
		return s
	}
	return "UNKNOWN"
}

// ParseMethod parses a string as an HTTPMethod.
func ParseMethod(method string) HTTPMethod {
	if m, ok := stringToMethod[strings.ToUpper(method)]; ok {
		return m
	}
	return MethodUnknown
}

// OptimizedTimings represents a memory-optimized timing structure.
type OptimizedTimings struct {
	Blocked         *float64 // Uses pointers to allow nil values.
	DNS             *float64 // Uses pointers to allow nil values.
	Connect         *float64 // Uses pointers to allow nil values.
	Send            *float64 // Uses pointers to allow nil values.
	Wait            *float64 // Uses pointers to allow nil values.
	Receive         *float64 // Uses pointers to allow nil values.
	Ssl             *float64 // Uses pointers to allow nil values.
	BlockedQueueing *float64 // Uses pointers to allow nil values.
	BlockedProxy    *float64 // Uses pointers to allow nil values.
}

// OptimizedContent 表示内存优化的内容结构
type OptimizedContent struct {
	Size     int     // 整数不需要优化
	MimeType string  // MIME类型通常不太长
	Text     *string // Uses pointers to allow nil values.
	Encoding *string // Uses pointers to allow nil values.
	Comment  *string // Uses pointers to allow nil values.
}

// OptimizedRequest 表示内存优化的请求结构
type OptimizedRequest struct {
	Method      HTTPMethod        // 使用枚举而不是字符串
	URL         string            // URL不能优化
	HTTPVersion string            // 版本号通常很短
	Cookies     []Cookie          // 保持不变
	Headers     map[string]string // 使用map而不是数组，优化查找
	QueryString map[string]string // 使用map而不是数组
	PostData    *PostData         // POST data.
	HeadersSize *int              // Uses pointers to allow nil values.
	BodySize    *int              // Uses pointers to allow nil values.
}

// OptimizedResponse 表示内存优化的响应结构
type OptimizedResponse struct {
	Status       int               // 整数不需要优化
	StatusText   string            // 状态文本通常很短
	HTTPVersion  string            // 版本号通常很短
	Cookies      []Cookie          // 保持不变
	Headers      map[string]string // 使用map而不是数组
	RedirectURL  string            // URL不能优化
	HeadersSize  *int              // Uses pointers to allow nil values.
	BodySize     *int              // Uses pointers to allow nil values.
	Content      *OptimizedContent // Uses pointers to allow nil values.
	TransferSize *int              // Uses pointers to allow nil values.
}

// OptimizedEntries 表示内存优化的条目结构
type OptimizedEntries struct {
	StartedDateTime time.Time         // 时间不需要优化
	Time            float64           // 浮点数不需要优化
	Request         OptimizedRequest  // 优化的请求
	Response        OptimizedResponse // 优化的响应
	Cache           *Cache            // Uses pointers to allow nil values.
	Timings         OptimizedTimings  // 优化的计时
	PageRef         *string           // Uses pointers to allow nil values.
	ServerIP        *string           // Uses pointers to allow nil values.
	Connection      *string           // Uses pointers to allow nil values.
}

// OptimizedHar 表示内存优化的HAR结构
type OptimizedHar struct {
	Log struct {
		Version string             // 版本号通常很短
		Creator Creator            // 保持不变
		Browser Browser            // 浏览器信息
		Pages   []Pages            // 保持不变
		Entries []OptimizedEntries // 优化的条目数组
	}
}

// ParseHarFileOptimized 解析HAR文件并返回内存优化的结构
func ParseHarFileOptimized(filePath string) (*OptimizedHar, error) {
	harFileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("failed to read HAR file '%s'", filePath), err)
	}

	return ParseHarOptimized(harFileBytes)
}

// ParseHarOptimized 解析HAR字节并返回内存优化的结构
func ParseHarOptimized(harFileBytes []byte) (*OptimizedHar, error) {
	// 先解析为标准HAR
	standardHar, err := ParseHar(harFileBytes)
	if err != nil {
		return nil, err
	}

	// 转换为优化的HAR
	optimizedHar := ToOptimizedHar(standardHar)
	return optimizedHar, nil
}

// ToOptimizedHar 将标准HAR转换为内存优化的HAR
func ToOptimizedHar(standardHar *Har) *OptimizedHar {
	if standardHar == nil {
		return nil
	}

	optimizedHar := &OptimizedHar{}
	optimizedHar.Log.Version = standardHar.Log.Version
	optimizedHar.Log.Creator = standardHar.Log.Creator
	optimizedHar.Log.Browser = standardHar.Log.Browser
	optimizedHar.Log.Pages = standardHar.Log.Pages

	// 转换所有条目
	optimizedHar.Log.Entries = make([]OptimizedEntries, len(standardHar.Log.Entries))
	for i, entry := range standardHar.Log.Entries {
		optimizedEntry := convertToOptimizedEntry(entry)
		optimizedHar.Log.Entries[i] = optimizedEntry
	}

	return optimizedHar
}

// convertToOptimizedEntry 将标准条目转换为优化条目
func convertToOptimizedEntry(entry Entries) OptimizedEntries {
	optimizedEntry := OptimizedEntries{
		StartedDateTime: entry.StartedDateTime,
		Time:            entry.Time,
	}

	// 转换请求
	optimizedEntry.Request = OptimizedRequest{
		Method:      ParseMethod(entry.Request.Method),
		URL:         entry.Request.URL,
		HTTPVersion: entry.Request.HTTPVersion,
		Cookies:     entry.Request.Cookies,
		Headers:     make(map[string]string, len(entry.Request.Headers)),
		QueryString: make(map[string]string, len(entry.Request.QueryString)),
		PostData:    entry.Request.PostData,
	}

	// 转换请求头
	for _, header := range entry.Request.Headers {
		optimizedEntry.Request.Headers[header.Name] = header.Value
	}

	// 转换查询参数
	for _, qs := range entry.Request.QueryString {
		optimizedEntry.Request.QueryString[qs.Name] = qs.Value
	}

	// 设置请求大小
	if entry.Request.HeadersSize != 0 {
		headerSize := entry.Request.HeadersSize
		optimizedEntry.Request.HeadersSize = &headerSize
	}
	if entry.Request.BodySize != 0 {
		bodySize := entry.Request.BodySize
		optimizedEntry.Request.BodySize = &bodySize
	}

	// Convert the response.
	optimizedEntry.Response = OptimizedResponse{
		Status:      entry.Response.Status,
		StatusText:  entry.Response.StatusText,
		HTTPVersion: entry.Response.HTTPVersion,
		Cookies:     entry.Response.Cookies,
		Headers:     make(map[string]string, len(entry.Response.Headers)),
		RedirectURL: entry.Response.RedirectURL,
	}

	// Convert response headers.
	for _, header := range entry.Response.Headers {
		optimizedEntry.Response.Headers[header.Name] = header.Value
	}

	// Set the response size.
	if entry.Response.HeadersSize != 0 {
		headerSize := entry.Response.HeadersSize
		optimizedEntry.Response.HeadersSize = &headerSize
	}
	if entry.Response.BodySize != 0 {
		bodySize := entry.Response.BodySize
		optimizedEntry.Response.BodySize = &bodySize
	}
	if entry.Response.TransferSize != 0 {
		transferSize := entry.Response.TransferSize
		optimizedEntry.Response.TransferSize = &transferSize
	}

	// Convert content.
	if entry.Response.Content.Size != 0 || entry.Response.Content.MimeType != "" ||
		entry.Response.Content.Text != "" || entry.Response.Content.Encoding != "" ||
		entry.Response.Content.Comment != "" {
		optimizedEntry.Response.Content = &OptimizedContent{
			Size:     entry.Response.Content.Size,
			MimeType: entry.Response.Content.MimeType,
		}
		if entry.Response.Content.Text != "" {
			text := entry.Response.Content.Text
			optimizedEntry.Response.Content.Text = &text
		}
		if entry.Response.Content.Encoding != "" {
			encoding := entry.Response.Content.Encoding
			optimizedEntry.Response.Content.Encoding = &encoding
		}
		if entry.Response.Content.Comment != "" {
			comment := entry.Response.Content.Comment
			optimizedEntry.Response.Content.Comment = &comment
		}
	}

	// Convert timings.
	if entry.Timings.Blocked != 0 {
		blocked := entry.Timings.Blocked
		optimizedEntry.Timings.Blocked = &blocked
	}
	if entry.Timings.DNS != 0 {
		dns := entry.Timings.DNS
		optimizedEntry.Timings.DNS = &dns
	}
	if entry.Timings.Connect != 0 {
		connect := entry.Timings.Connect
		optimizedEntry.Timings.Connect = &connect
	}
	if entry.Timings.Send != 0 {
		send := entry.Timings.Send
		optimizedEntry.Timings.Send = &send
	}
	if entry.Timings.Wait != 0 {
		wait := entry.Timings.Wait
		optimizedEntry.Timings.Wait = &wait
	}
	if entry.Timings.Receive != 0 {
		receive := entry.Timings.Receive
		optimizedEntry.Timings.Receive = &receive
	}
	if entry.Timings.Ssl != 0 {
		ssl := entry.Timings.Ssl
		optimizedEntry.Timings.Ssl = &ssl
	}
	if entry.Timings.BlockedQueueing != 0 {
		blockedQueueing := entry.Timings.BlockedQueueing
		optimizedEntry.Timings.BlockedQueueing = &blockedQueueing
	}
	if entry.Timings.BlockedProxy != 0 {
		blockedProxy := entry.Timings.BlockedProxy
		optimizedEntry.Timings.BlockedProxy = &blockedProxy
	}

	// Convert cache data.
	if entry.Cache.Comment != "" ||
		entry.Cache.BeforeRequest != nil ||
		entry.Cache.AfterRequest != nil {
		cache := entry.Cache
		optimizedEntry.Cache = &cache
	}

	// Set optional fields.
	if entry.Pageref != "" {
		pageRef := entry.Pageref
		optimizedEntry.PageRef = &pageRef
	}
	if entry.ServerIPAddress != "" {
		serverIP := entry.ServerIPAddress
		optimizedEntry.ServerIP = &serverIP
	}
	if entry.Connection != "" {
		connection := entry.Connection
		optimizedEntry.Connection = &connection
	}

	return optimizedEntry
}

// ToStandardHar 将优化的HAR转换回标准HAR
func (oh *OptimizedHar) ToStandardHar() *Har {
	if oh == nil {
		return nil
	}

	standardHar := &Har{}
	standardHar.Log.Version = oh.Log.Version
	standardHar.Log.Creator = oh.Log.Creator
	standardHar.Log.Browser = oh.Log.Browser
	standardHar.Log.Pages = oh.Log.Pages

	// 转换所有条目
	standardHar.Log.Entries = make([]Entries, len(oh.Log.Entries))
	for i, entry := range oh.Log.Entries {
		standardHar.Log.Entries[i] = convertToStandardEntry(entry)
	}

	return standardHar
}

// convertToStandardEntry 将优化条目转换为标准条目
func convertToStandardEntry(entry OptimizedEntries) Entries {
	standardEntry := Entries{
		StartedDateTime: entry.StartedDateTime,
		Time:            entry.Time,
	}

	// 转换请求
	standardEntry.Request = Request{
		Method:      entry.Request.Method.String(),
		URL:         entry.Request.URL,
		HTTPVersion: entry.Request.HTTPVersion,
		Cookies:     entry.Request.Cookies,
		Headers:     make([]Headers, 0, len(entry.Request.Headers)),
		QueryString: make([]QueryString, 0, len(entry.Request.QueryString)),
		PostData:    entry.Request.PostData,
	}

	// 转换请求头
	for name, value := range entry.Request.Headers {
		standardEntry.Request.Headers = append(standardEntry.Request.Headers, Headers{
			Name:  name,
			Value: value,
		})
	}

	// 转换查询参数
	for name, value := range entry.Request.QueryString {
		standardEntry.Request.QueryString = append(standardEntry.Request.QueryString, QueryString{
			Name:  name,
			Value: value,
		})
	}

	// 设置请求大小
	if entry.Request.HeadersSize != nil {
		standardEntry.Request.HeadersSize = *entry.Request.HeadersSize
	}
	if entry.Request.BodySize != nil {
		standardEntry.Request.BodySize = *entry.Request.BodySize
	}

	// Convert the response.
	standardEntry.Response = Response{
		Status:      entry.Response.Status,
		StatusText:  entry.Response.StatusText,
		HTTPVersion: entry.Response.HTTPVersion,
		Cookies:     entry.Response.Cookies,
		Headers:     make([]Headers, 0, len(entry.Response.Headers)),
		RedirectURL: entry.Response.RedirectURL,
	}

	// Convert response headers.
	for name, value := range entry.Response.Headers {
		standardEntry.Response.Headers = append(standardEntry.Response.Headers, Headers{
			Name:  name,
			Value: value,
		})
	}

	// Set the response size.
	if entry.Response.HeadersSize != nil {
		standardEntry.Response.HeadersSize = *entry.Response.HeadersSize
	}
	if entry.Response.BodySize != nil {
		standardEntry.Response.BodySize = *entry.Response.BodySize
	}
	if entry.Response.TransferSize != nil {
		standardEntry.Response.TransferSize = *entry.Response.TransferSize
	}

	// Convert content.
	if entry.Response.Content != nil {
		standardEntry.Response.Content = Content{
			Size:     entry.Response.Content.Size,
			MimeType: entry.Response.Content.MimeType,
		}
		if entry.Response.Content.Text != nil {
			standardEntry.Response.Content.Text = *entry.Response.Content.Text
		}
		if entry.Response.Content.Encoding != nil {
			standardEntry.Response.Content.Encoding = *entry.Response.Content.Encoding
		}
		if entry.Response.Content.Comment != nil {
			standardEntry.Response.Content.Comment = *entry.Response.Content.Comment
		}
	}

	// Convert timings.
	if entry.Timings.Blocked != nil {
		standardEntry.Timings.Blocked = *entry.Timings.Blocked
	}
	if entry.Timings.DNS != nil {
		standardEntry.Timings.DNS = *entry.Timings.DNS
	}
	if entry.Timings.Connect != nil {
		standardEntry.Timings.Connect = *entry.Timings.Connect
	}
	if entry.Timings.Send != nil {
		standardEntry.Timings.Send = *entry.Timings.Send
	}
	if entry.Timings.Wait != nil {
		standardEntry.Timings.Wait = *entry.Timings.Wait
	}
	if entry.Timings.Receive != nil {
		standardEntry.Timings.Receive = *entry.Timings.Receive
	}
	if entry.Timings.Ssl != nil {
		standardEntry.Timings.Ssl = *entry.Timings.Ssl
	}
	if entry.Timings.BlockedQueueing != nil {
		standardEntry.Timings.BlockedQueueing = *entry.Timings.BlockedQueueing
	}
	if entry.Timings.BlockedProxy != nil {
		standardEntry.Timings.BlockedProxy = *entry.Timings.BlockedProxy
	}

	// Convert cache data.
	if entry.Cache != nil {
		standardEntry.Cache = *entry.Cache
	}

	// Set optional fields.
	if entry.PageRef != nil {
		standardEntry.Pageref = *entry.PageRef
	}
	if entry.ServerIP != nil {
		standardEntry.ServerIPAddress = *entry.ServerIP
	}
	if entry.Connection != nil {
		standardEntry.Connection = *entry.Connection
	}

	return standardEntry
}

// SearchByURL searches entries by URL.
func (oh *OptimizedHar) SearchByURL(urlPattern string) []OptimizedEntries {
	if oh == nil {
		return nil
	}

	var results []OptimizedEntries

	for _, entry := range oh.Log.Entries {
		if strings.Contains(entry.Request.URL, urlPattern) {
			results = append(results, entry)
		}
	}

	return results
}

// SearchByMethod searches entries by HTTP method.
func (oh *OptimizedHar) SearchByMethod(method HTTPMethod) []OptimizedEntries {
	if oh == nil {
		return nil
	}

	var results []OptimizedEntries

	for _, entry := range oh.Log.Entries {
		if entry.Request.Method == method {
			results = append(results, entry)
		}
	}

	return results
}

// SearchByStatusCode searches entries by status code.
func (oh *OptimizedHar) SearchByStatusCode(statusCode int) []OptimizedEntries {
	if oh == nil {
		return nil
	}

	var results []OptimizedEntries

	for _, entry := range oh.Log.Entries {
		if entry.Response.Status == statusCode {
			results = append(results, entry)
		}
	}

	return results
}

// GetRequestHeaderValue returns the value of the specified request header.
func (req *OptimizedRequest) GetRequestHeaderValue(name string) (string, bool) {
	if req == nil {
		return "", false
	}

	value, ok := req.Headers[name]
	return value, ok
}

// GetResponseHeaderValue returns the value of the specified response header.
func (resp *OptimizedResponse) GetResponseHeaderValue(name string) (string, bool) {
	if resp == nil {
		return "", false
	}

	value, ok := resp.Headers[name]
	return value, ok
}

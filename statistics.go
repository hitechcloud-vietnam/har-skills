package har

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// HarStatistics contains statistics about a HAR file.
type HarStatistics struct {
	TotalRequests     int            // Total request count.
	TotalTransferred  int64          // Total transferred bytes.
	TotalUncompressed int64          // Total uncompressed bytes.
	TotalTime         float64        // Total time (ms) from the first request to the last response.
	AvgTime           float64        // Average request duration (ms).
	MaxTime           float64        // Maximum request duration (ms).
	MinTime           float64        // Minimum request duration (ms).
	MedianTime        float64        // Median request duration (ms).
	P95Time           float64        // 95th-percentile request duration (ms).
	P99Time           float64        // 99th-percentile request duration (ms).
	Methods           map[string]int // HTTP method distribution.
	StatusCodes       map[int]int    // Status-code distribution.
	ContentTypes      map[string]int // Content-type distribution.
	Domains           map[string]int // Domain distribution.
	ErrorCount        int            // Number of failed requests (4xx+5xx).
	RedirectCount     int            // Number of redirects (3xx).
	TimingsSummary    TimingsSummary // Timing summary.
	StartTime         time.Time      // Earliest request time.
	EndTime           time.Time      // Latest request time.
}

// TimingsSummary contains aggregated timing statistics.
type TimingsSummary struct {
	AvgBlocked float64 // Average blocked time (ms).
	AvgDNS     float64 // Average DNS resolution time (ms).
	AvgConnect float64 // Average TCP connection time (ms).
	AvgSend    float64 // Average send time (ms).
	AvgWait    float64 // Average wait time (ms).
	AvgReceive float64 // Average receive time (ms).
	AvgSSL     float64 // Average SSL handshake time (ms).
	MaxBlocked float64 // Maximum blocked time (ms).
	MaxDNS     float64 // Maximum DNS resolution time (ms).
	MaxConnect float64 // Maximum TCP connection time (ms).
	MaxSend    float64 // Maximum send time (ms).
	MaxWait    float64 // Maximum wait time (ms).
	MaxReceive float64 // Maximum receive time (ms).
	MaxSSL     float64 // Maximum SSL handshake time (ms).
	MinBlocked float64 // Minimum blocked time (ms).
	MinDNS     float64 // Minimum DNS resolution time (ms).
	MinConnect float64 // Minimum TCP connection time (ms).
	MinSend    float64 // Minimum send time (ms).
	MinWait    float64 // Minimum wait time (ms).
	MinReceive float64 // Minimum receive time (ms).
	MinSSL     float64 // Minimum SSL handshake time (ms).
}

// DomainStats contains statistics grouped by domain.
type DomainStats struct {
	RequestCount     int     // Request count.
	TotalTime        float64 // Total duration (ms).
	AvgTime          float64 // Average duration (ms).
	TotalTransferred int64   // Total transferred bytes.
	ErrorCount       int     // Error count.
}

// Statistics calculates full statistics for a HAR file.
func (h *Har) Statistics() *HarStatistics {
	if h == nil || len(h.Log.Entries) == 0 {
		return &HarStatistics{
			Methods:      make(map[string]int),
			StatusCodes:  make(map[int]int),
			ContentTypes: make(map[string]int),
			Domains:      make(map[string]int),
		}
	}

	stats := &HarStatistics{
		TotalRequests: len(h.Log.Entries),
		Methods:       make(map[string]int),
		StatusCodes:   make(map[int]int),
		ContentTypes:  make(map[string]int),
		Domains:       make(map[string]int),
	}

	var totalTime float64
	var times []float64
	var startTime, endTime time.Time
	var validTimings int
	var sumBlocked, sumDNS, sumConnect, sumSend, sumWait, sumReceive, sumSSL float64
	var countBlocked, countDNS, countConnect, countSend, countWait, countReceive, countSSL int

	// Initialize the minimum to a large value.
	minTime := float64(1 << 62)
	var minBlocked, minDNS, minConnect, minSend, minWait, minReceive, minSSL float64 = 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62

	for i, entry := range h.Log.Entries {
		// Accumulate total time.
		totalTime += entry.Time
		times = append(times, entry.Time)

		// Minimum/maximum request duration.
		if entry.Time > stats.MaxTime {
			stats.MaxTime = entry.Time
		}
		if entry.Time < minTime {
			minTime = entry.Time
		}

		// Transferred bytes.
		if entry.Response.BodySize > 0 {
			stats.TotalTransferred += int64(entry.Response.BodySize)
		}
		if entry.Response.Content.Size > 0 {
			stats.TotalUncompressed += int64(entry.Response.Content.Size)
		}

		// HTTP method distribution.
		stats.Methods[entry.Request.Method]++

		// Status-code distribution.
		stats.StatusCodes[entry.Response.Status]++

		// Error count.
		if entry.Response.Status >= 400 {
			stats.ErrorCount++
		}
		if entry.Response.Status >= 300 && entry.Response.Status < 400 {
			stats.RedirectCount++
		}

		// Content-type distribution.
		contentType := entry.Response.Content.MimeType
		if contentType != "" {
			// Remove parameters (such as charset=utf-8).
			if idx := strings.Index(contentType, ";"); idx != -1 {
				contentType = strings.TrimSpace(contentType[:idx])
			}
			stats.ContentTypes[contentType]++
		}

		// Domain distribution.
		if domain := extractDomain(entry.Request.URL); domain != "" {
			stats.Domains[domain]++
		}

		// Time range.
		if i == 0 || entry.StartedDateTime.Before(startTime) {
			startTime = entry.StartedDateTime
		}
		if endTime.IsZero() || entry.StartedDateTime.Add(time.Duration(entry.Time)*time.Millisecond).After(endTime) {
			endTime = entry.StartedDateTime.Add(time.Duration(entry.Time) * time.Millisecond)
		}

		// Timing summary.
		if entry.Timings.Blocked > 0 {
			sumBlocked += entry.Timings.Blocked
			countBlocked++
			if entry.Timings.Blocked > stats.TimingsSummary.MaxBlocked {
				stats.TimingsSummary.MaxBlocked = entry.Timings.Blocked
			}
			if entry.Timings.Blocked < minBlocked {
				minBlocked = entry.Timings.Blocked
			}
		}
		if entry.Timings.DNS > 0 {
			sumDNS += entry.Timings.DNS
			countDNS++
			if entry.Timings.DNS > stats.TimingsSummary.MaxDNS {
				stats.TimingsSummary.MaxDNS = entry.Timings.DNS
			}
			if entry.Timings.DNS < minDNS {
				minDNS = entry.Timings.DNS
			}
		}
		if entry.Timings.Connect > 0 {
			sumConnect += entry.Timings.Connect
			countConnect++
			if entry.Timings.Connect > stats.TimingsSummary.MaxConnect {
				stats.TimingsSummary.MaxConnect = entry.Timings.Connect
			}
			if entry.Timings.Connect < minConnect {
				minConnect = entry.Timings.Connect
			}
		}
		if entry.Timings.Send > 0 {
			sumSend += entry.Timings.Send
			countSend++
			if entry.Timings.Send > stats.TimingsSummary.MaxSend {
				stats.TimingsSummary.MaxSend = entry.Timings.Send
			}
			if entry.Timings.Send < minSend {
				minSend = entry.Timings.Send
			}
		}
		if entry.Timings.Wait > 0 {
			sumWait += entry.Timings.Wait
			countWait++
			if entry.Timings.Wait > stats.TimingsSummary.MaxWait {
				stats.TimingsSummary.MaxWait = entry.Timings.Wait
			}
			if entry.Timings.Wait < minWait {
				minWait = entry.Timings.Wait
			}
		}
		if entry.Timings.Receive > 0 {
			sumReceive += entry.Timings.Receive
			countReceive++
			if entry.Timings.Receive > stats.TimingsSummary.MaxReceive {
				stats.TimingsSummary.MaxReceive = entry.Timings.Receive
			}
			if entry.Timings.Receive < minReceive {
				minReceive = entry.Timings.Receive
			}
		}
		if entry.Timings.Ssl > 0 {
			sumSSL += entry.Timings.Ssl
			countSSL++
			if entry.Timings.Ssl > stats.TimingsSummary.MaxSSL {
				stats.TimingsSummary.MaxSSL = entry.Timings.Ssl
			}
			if entry.Timings.Ssl < minSSL {
				minSSL = entry.Timings.Ssl
			}
		}
		validTimings++
	}

	// Calculate averages.
	stats.AvgTime = totalTime / float64(stats.TotalRequests)
	stats.MinTime = minTime

	// Calculate percentiles.
	stats.MedianTime = percentile(times, 50)
	stats.P95Time = percentile(times, 95)
	stats.P99Time = percentile(times, 99)

	// Total time range.
	if !startTime.IsZero() && !endTime.IsZero() {
		stats.TotalTime = float64(endTime.Sub(startTime).Milliseconds())
		stats.StartTime = startTime
		stats.EndTime = endTime
	}

	// Calculate average timing metrics.
	if validTimings > 0 {
		n := float64(validTimings)
		stats.TimingsSummary.AvgBlocked = sumBlocked / n
		stats.TimingsSummary.AvgDNS = sumDNS / n
		stats.TimingsSummary.AvgConnect = sumConnect / n
		stats.TimingsSummary.AvgSend = sumSend / n
		stats.TimingsSummary.AvgWait = sumWait / n
		stats.TimingsSummary.AvgReceive = sumReceive / n
		stats.TimingsSummary.AvgSSL = sumSSL / n
	}

	// Set the minimum if a valid value exists.
	if minBlocked < float64(1<<62) {
		stats.TimingsSummary.MinBlocked = minBlocked
	}
	if minDNS < float64(1<<62) {
		stats.TimingsSummary.MinDNS = minDNS
	}
	if minConnect < float64(1<<62) {
		stats.TimingsSummary.MinConnect = minConnect
	}
	if minSend < float64(1<<62) {
		stats.TimingsSummary.MinSend = minSend
	}
	if minWait < float64(1<<62) {
		stats.TimingsSummary.MinWait = minWait
	}
	if minReceive < float64(1<<62) {
		stats.TimingsSummary.MinReceive = minReceive
	}
	if minSSL < float64(1<<62) {
		stats.TimingsSummary.MinSSL = minSSL
	}

	return stats
}

// TimingStatistics calculates timing statistics only.
func (h *Har) TimingStatistics() *TimingsSummary {
	stats := h.Statistics()
	return &stats.TimingsSummary
}

// DomainSummary returns statistics grouped by domain.
func (h *Har) DomainSummary() map[string]*DomainStats {
	result := make(map[string]*DomainStats)

	if h == nil {
		return result
	}

	for _, entry := range h.Log.Entries {
		domain := extractDomain(entry.Request.URL)
		if domain == "" {
			continue
		}

		if _, ok := result[domain]; !ok {
			result[domain] = &DomainStats{}
		}

		ds := result[domain]
		ds.RequestCount++
		ds.TotalTime += entry.Time

		if entry.Response.BodySize > 0 {
			ds.TotalTransferred += int64(entry.Response.BodySize)
		}

		if entry.Response.Status >= 400 {
			ds.ErrorCount++
		}
	}

	// Calculate the average duration.
	for _, ds := range result {
		if ds.RequestCount > 0 {
			ds.AvgTime = ds.TotalTime / float64(ds.RequestCount)
		}
	}

	return result
}

// StatusCodeDistribution returns the status-code distribution.
func (h *Har) StatusCodeDistribution() map[int]int {
	if h == nil {
		return make(map[int]int)
	}

	result := make(map[int]int)
	for _, entry := range h.Log.Entries {
		result[entry.Response.Status]++
	}
	return result
}

// MethodDistribution returns the HTTP method distribution.
func (h *Har) MethodDistribution() map[string]int {
	if h == nil {
		return make(map[string]int)
	}

	result := make(map[string]int)
	for _, entry := range h.Log.Entries {
		result[entry.Request.Method]++
	}
	return result
}

// ContentTypeDistribution returns the content-type distribution.
func (h *Har) ContentTypeDistribution() map[string]int {
	if h == nil {
		return make(map[string]int)
	}

	result := make(map[string]int)
	for _, entry := range h.Log.Entries {
		contentType := entry.Response.Content.MimeType
		if contentType != "" {
			if idx := strings.Index(contentType, ";"); idx != -1 {
				contentType = strings.TrimSpace(contentType[:idx])
			}
			result[contentType]++
		}
	}
	return result
}

// SlowestRequests returns the N slowest requests.
func (h *Har) SlowestRequests(n int) []Entries {
	if h == nil || n <= 0 {
		return nil
	}

	entries := make([]Entries, len(h.Log.Entries))
	copy(entries, h.Log.Entries)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time > entries[j].Time
	})

	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

// FastestRequests returns the N fastest requests.
func (h *Har) FastestRequests(n int) []Entries {
	if h == nil || n <= 0 {
		return nil
	}

	entries := make([]Entries, len(h.Log.Entries))
	copy(entries, h.Log.Entries)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time < entries[j].Time
	})

	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

// LargestResponses returns the N requests with the largest response bodies.
func (h *Har) LargestResponses(n int) []Entries {
	if h == nil || n <= 0 {
		return nil
	}

	entries := make([]Entries, len(h.Log.Entries))
	copy(entries, h.Log.Entries)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Response.Content.Size > entries[j].Response.Content.Size
	})

	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

// Summary returns a text summary of a HAR file.
func (h *Har) Summary() string {
	stats := h.Statistics()
	var log Log
	if h != nil {
		log = h.Log
	}

	var sb strings.Builder
	sb.WriteString("HAR File Summary\n")
	sb.WriteString("=============\n")
	sb.WriteString(fmt.Sprintf("Version: %s\n", log.Version))
	if log.Creator.Name != "" {
		sb.WriteString(fmt.Sprintf("Creator: %s %s\n", log.Creator.Name, log.Creator.Version))
	}
	if log.Browser.Name != "" {
		sb.WriteString(fmt.Sprintf("Browser: %s %s\n", log.Browser.Name, log.Browser.Version))
	}
	sb.WriteString(fmt.Sprintf("Total requests: %d\n", stats.TotalRequests))
	sb.WriteString(fmt.Sprintf("Failed requests: %d\n", stats.ErrorCount))
	sb.WriteString(fmt.Sprintf("Redirects: %d\n", stats.RedirectCount))
	sb.WriteString(fmt.Sprintf("Total transferred: %s\n", formatBytes(stats.TotalTransferred)))
	sb.WriteString(fmt.Sprintf("Total uncompressed: %s\n", formatBytes(stats.TotalUncompressed)))
	sb.WriteString(fmt.Sprintf("Total time: %.2f ms\n", stats.TotalTime))
	sb.WriteString(fmt.Sprintf("Average request duration: %.2f ms\n", stats.AvgTime))
	sb.WriteString(fmt.Sprintf("Median request duration: %.2f ms\n", stats.MedianTime))
	sb.WriteString(fmt.Sprintf("P95 request duration: %.2f ms\n", stats.P95Time))
	sb.WriteString(fmt.Sprintf("P99 request duration: %.2f ms\n", stats.P99Time))
	sb.WriteString(fmt.Sprintf("Slowest request: %.2f ms\n", stats.MaxTime))
	sb.WriteString(fmt.Sprintf("Fastest request: %.2f ms\n", stats.MinTime))

	if len(stats.Domains) > 0 {
		sb.WriteString(fmt.Sprintf("\nDomain count: %d\n", len(stats.Domains)))
	}

	return sb.String()
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// percentile calculates a percentile.
func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	index := float64(p) / 100.0 * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	fraction := index - float64(lower)
	return sorted[lower]*(1-fraction) + sorted[upper]*fraction
}

// formatBytes formats bytes as a human-readable string (uses FormatBytes internally).
func formatBytes(bytes int64) string {
	return FormatBytes(int(bytes))
}

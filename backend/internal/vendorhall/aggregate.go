package vendorhall

import (
	"math"
	"sort"
	"strconv"
	"time"
)

func ptrFloat(value float64) *float64 { return &value }

func histogramP95(histogram map[string]int64) *float64 {
	type entry struct {
		value float64
		count int64
	}
	entries := make([]entry, 0, len(histogram))
	var total int64
	for bucket, count := range histogram {
		value, err := strconv.ParseFloat(bucket, 64)
		if err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) && count > 0 {
			entries = append(entries, entry{value, count})
			total += count
		}
	}
	if total == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].value < entries[j].value })
	target := int64(math.Ceil(float64(total) * .95))
	var seen int64
	for _, item := range entries {
		seen += item.count
		if seen >= target {
			return ptrFloat(item.value)
		}
	}
	return ptrFloat(entries[len(entries)-1].value)
}

func histogramAverage(histogram map[string]int64) *float64 {
	var totalCount int64
	var weightedSum float64
	for bucket, count := range histogram {
		value, err := strconv.ParseFloat(bucket, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || count <= 0 {
			continue
		}
		totalCount += count
		weightedSum += value * float64(count)
	}
	if totalCount == 0 {
		return nil
	}
	return ptrFloat(weightedSum / float64(totalCount))
}

func aggregateMinutes(minutes []MinuteMetric, start, end time.Time, maxTrendPoints int) Metrics {
	result := Metrics{Trend: []TrendPoint{}, hasSamples: len(minutes) > 0}
	durationHist := map[string]int64{}
	firstTokenHist := map[string]int64{}
	var durationCount, durationSum, input, cacheRead, cacheCreate int64
	var latestMultiplierAt time.Time
	var latestBalanceAt time.Time
	for _, row := range minutes {
		if result.collectedAt == nil || row.BucketStart.After(*result.collectedAt) {
			value := row.BucketStart.UTC()
			result.collectedAt = &value
		}
		result.SuccessCount += row.SuccessCount
		result.UpstreamErrorCount += row.UpstreamErrorCount
		result.EligibleCount += row.EligibleCount
		durationCount += row.DurationCount
		durationSum += row.DurationSumMs
		input += row.InputTokens
		cacheRead += row.CacheReadTokens
		cacheCreate += row.CacheCreationTokens
		for key, count := range row.DurationHistogram {
			durationHist[key] += count
		}
		for key, count := range row.FirstTokenHistogram {
			firstTokenHist[key] += count
		}
		if latestMultiplierAt.IsZero() || row.BucketStart.After(latestMultiplierAt) {
			result.UpstreamMultiplier = row.UpstreamMultiplier
			latestMultiplierAt = row.BucketStart
		}
		if latestBalanceAt.IsZero() || row.BucketStart.After(latestBalanceAt) {
			result.BalanceUSD = row.BalanceUSD
			latestBalanceAt = row.BucketStart
		}
	}
	result.RequestCount = result.SuccessCount + result.UpstreamErrorCount
	if result.EligibleCount > 0 {
		result.Availability = ptrFloat(float64(result.SuccessCount) / float64(result.EligibleCount))
	}
	denominator := input + cacheRead + cacheCreate
	if denominator > 0 {
		result.CacheHitRate = ptrFloat(float64(cacheRead) / float64(denominator))
	}
	if durationCount > 0 {
		result.AverageDurationMs = ptrFloat(float64(durationSum) / float64(durationCount))
	}
	result.DurationP95Ms = histogramP95(durationHist)
	result.FirstTokenP95Ms = histogramP95(firstTokenHist)
	result.FirstTokenAverageMs = histogramAverage(firstTokenHist)
	result.inputTokens = input
	result.cacheReadTokens = cacheRead
	result.cacheCreateTokens = cacheCreate
	result.firstTokenHistogram = firstTokenHist
	if maxTrendPoints <= 0 || len(minutes) == 0 {
		return result
	}
	bucketWidth := end.Sub(start) / time.Duration(maxTrendPoints)
	if bucketWidth < time.Minute {
		bucketWidth = time.Minute
	}
	type trendAgg struct {
		first map[string]int64
	}
	buckets := map[int64]*trendAgg{}
	for _, row := range minutes {
		index := int64(row.BucketStart.Sub(start) / bucketWidth)
		if index < 0 {
			continue
		}
		if index >= int64(maxTrendPoints) {
			index = int64(maxTrendPoints - 1)
		}
		agg := buckets[index]
		if agg == nil {
			agg = &trendAgg{first: map[string]int64{}}
			buckets[index] = agg
		}
		for key, count := range row.FirstTokenHistogram {
			agg.first[key] += count
		}
	}
	indexes := make([]int, 0, len(buckets))
	for index := range buckets {
		indexes = append(indexes, int(index))
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		agg := buckets[int64(index)]
		result.Trend = append(result.Trend, TrendPoint{Timestamp: start.Add(time.Duration(index) * bucketWidth), TTFTP95Ms: histogramP95(agg.first)})
	}
	return result
}

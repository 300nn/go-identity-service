package metrics

type CacheObserver struct {
	metrics *Metrics
}

func NewCacheObserver(metrics *Metrics) *CacheObserver {
	return &CacheObserver{metrics: metrics}
}

func (o *CacheObserver) Hit(cacheName string) {
	o.metrics.RedisCacheHitsTotal.WithLabelValues(cacheName).Inc()
}

func (o *CacheObserver) Miss(cacheName string) {
	o.metrics.RedisCacheMissesTotal.WithLabelValues(cacheName).Inc()
}

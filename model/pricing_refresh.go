package model

// RefreshEndpointSupportCache 强制立即重新计算模型端点支持缓存。
func RefreshEndpointSupportCache() {
	updateEndpointSupportLock.Lock()
	defer updateEndpointSupportLock.Unlock()
	modelSupportEndpointsLock.Lock()
	defer modelSupportEndpointsLock.Unlock()

	rebuildEndpointSupportCache()
}

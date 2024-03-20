package repo

// we set expire time to indicate real stats of edges
const (
	frontiersKeyPrefix    = "frontlas:frontiers:" // example: frontlas:alive:frontiers:123 "{}"
	frontiersKeyPrefixAll = "frontlas:frontiers:*"

	frontiersAliveKeyPrefix = "frontlas:alive:frontiers:" // example: frontlas:alive:frontiers:123 1 ex 20
	// TODO take care of reboot of redis
)

func getFrontierKey(frontier string) string {
	return frontiersKeyPrefix + frontier
}

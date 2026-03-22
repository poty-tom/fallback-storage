package storage

// Location はデータがどのストレージに存在するかを表す所在情報
type Location int

const (
	LocationPrimary Location = iota
	LocationFallback
)

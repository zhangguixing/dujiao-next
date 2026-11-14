package domain

// HomeVisitor retains one anonymous identifier per storefront.
type HomeVisitor struct {
	ResellerID uint   `gorm:"primaryKey;autoIncrement:false"`
	VisitorKey string `gorm:"size:64;primaryKey"`
	VisitDay   string `gorm:"size:10;not null;index"`
}

type HomeVisitorStats struct {
	Today int64 `json:"today"`
	Total int64 `json:"total"`
}

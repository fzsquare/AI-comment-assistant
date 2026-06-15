package model

import "time"

const (
	StatusDisabled = 0
	StatusEnabled  = 1
)

const (
	ReviewStatusAvailable = "available"
	ReviewStatusDeleted   = "deleted"
	ReviewStatusDisabled  = "disabled"
	ReviewStatusPending   = "pending_review"
)

const (
	TaskStatusPending       = "pending"
	TaskStatusRunning       = "running"
	TaskStatusSuccess       = "success"
	TaskStatusPartialFailed = "partial_failed"
	TaskStatusFailed        = "failed"
)

const (
	TriggerInit       = "init"
	TriggerManual     = "manual"
	TriggerAutoRefill = "auto_refill"
)

const (
	TagStatusUnbound  = "unbound"
	TagStatusBound    = "bound"
	TagStatusDisabled = "disabled"
)

type AdminUser struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Account      string    `gorm:"size:64;uniqueIndex;not null" json:"account"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Status       int       `gorm:"default:1;not null" json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type MerchantUser struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Account      string    `gorm:"size:64;uniqueIndex;not null" json:"account"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	MerchantName string    `gorm:"size:128;not null" json:"merchantName"`
	ContactName  string    `gorm:"size:128;not null" json:"contactName"`
	Status       int       `gorm:"default:1;not null" json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Store struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	MerchantUserID       uint      `gorm:"uniqueIndex;not null" json:"merchantUserId"`
	StoreName            string    `gorm:"size:128;not null" json:"storeName"`
	IndustryType         string    `gorm:"size:64" json:"industryType"`
	StoreIntro           string    `gorm:"type:text" json:"storeIntro"`
	Address              string    `gorm:"size:255" json:"address"`
	PrimaryPlatformStyle string    `gorm:"size:64;not null" json:"primaryPlatformStyle"`
	BrandTone            string    `gorm:"size:255" json:"brandTone"`
	Status               int       `gorm:"default:1;not null" json:"status"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type StoreKeyword struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	StoreID   uint      `gorm:"index;not null" json:"storeId"`
	Keyword   string    `gorm:"size:128;not null" json:"keyword"`
	SortNo    int       `gorm:"default:0" json:"sortNo"`
	CreatedAt time.Time `json:"createdAt"`
}

type StoreImage struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	StoreID      uint      `gorm:"index;not null" json:"storeId"`
	ImageURL     string    `gorm:"size:500;not null" json:"imageUrl"`
	ThumbnailURL string    `gorm:"size:500" json:"thumbnailUrl"`
	Status       int       `gorm:"default:1;not null" json:"status"`
	SortNo       int       `gorm:"default:0" json:"sortNo"`
	CreatedAt    time.Time `json:"createdAt"`
}

type StorePlatformLink struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	StoreID      uint      `gorm:"index;not null" json:"storeId"`
	PlatformCode string    `gorm:"size:64;not null" json:"platformCode"`
	PlatformName string    `gorm:"size:128;not null" json:"platformName"`
	ButtonText   string    `gorm:"size:128;not null" json:"buttonText"`
	TargetURL    string    `gorm:"size:500;not null" json:"targetUrl"`
	BackupURL    string    `gorm:"size:500" json:"backupUrl"`
	SortNo       int       `gorm:"default:0" json:"sortNo"`
	Status       int       `gorm:"default:1;not null" json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ReviewItem struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	StoreID           uint       `gorm:"index;not null" json:"storeId"`
	PlatformStyle     string     `gorm:"size:64;not null" json:"platformStyle"`
	Content           string     `gorm:"type:text;not null" json:"content"`
	SourceType        string     `gorm:"size:32;not null" json:"sourceType"`
	GenerationBatchNo string     `gorm:"size:64;not null" json:"generationBatchNo"`
	IsDispatched      bool       `gorm:"default:false;not null" json:"isDispatched"`
	Status            string     `gorm:"size:32;default:'available';not null" json:"status"`
	DispatchedAt      *time.Time `json:"dispatchedAt"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type ReviewDisplayLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	StoreID      uint      `gorm:"index;not null" json:"storeId"`
	ReviewItemID uint      `gorm:"index" json:"reviewItemId"`
	NFCTagID     uint      `gorm:"index" json:"nfcTagId"`
	SessionID    string    `gorm:"size:128;index;not null" json:"sessionId"`
	ActionType   string    `gorm:"size:64;not null" json:"actionType"`
	PlatformCode string    `gorm:"size:64" json:"platformCode"`
	ClientIP     string    `gorm:"size:64" json:"clientIp"`
	UserAgent    string    `gorm:"size:255" json:"userAgent"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ReviewGenerationTask struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	StoreID       uint      `gorm:"index;not null" json:"storeId"`
	PlatformStyle string    `gorm:"size:64;not null" json:"platformStyle"`
	TriggerType   string    `gorm:"size:32;not null" json:"triggerType"`
	TargetCount   int       `gorm:"not null" json:"targetCount"`
	SuccessCount  int       `gorm:"default:0;not null" json:"successCount"`
	FailedCount   int       `gorm:"default:0;not null" json:"failedCount"`
	Status        string    `gorm:"size:32;default:'pending';not null" json:"status"`
	ErrorMessage  string    `gorm:"type:text" json:"errorMessage"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type NFCTag struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TagCode      string    `gorm:"size:128;uniqueIndex;not null" json:"tagCode"`
	StoreID      uint      `gorm:"index" json:"storeId"`
	LandingToken string    `gorm:"size:128;uniqueIndex;not null" json:"landingToken"`
	Status       string    `gorm:"size:32;default:'unbound';not null" json:"status"`
	Remark       string    `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

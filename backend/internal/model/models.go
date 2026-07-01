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
	ReviewFeedbackAccepted = "accepted"
	ReviewFeedbackRejected = "rejected"
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

// StoreType 门店类型标签：预置 9 行业 + 自定义。IndustryCode 为生成/隔离基准
// （对应 agent-service 的 9 行业 code）。
type StoreType struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Code         string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name         string    `gorm:"size:64;not null" json:"name"`
	IndustryCode string    `gorm:"size:64;not null;default:'restaurant'" json:"industryCode"`
	IsPreset     bool      `gorm:"default:false;not null" json:"isPreset"`
	Status       int       `gorm:"default:1;not null" json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Store struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	MerchantUserID       uint      `gorm:"uniqueIndex;not null" json:"merchantUserId"`
	UUID                 string    `gorm:"size:36;uniqueIndex;not null" json:"uuid"`
	TypeID               *uint     `gorm:"index" json:"typeId"`
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
	Tags              string     `gorm:"size:255;default:''" json:"tags"`
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
	StoreID      uint      `gorm:"index;index:idx_review_logs_store_action_created,priority:1;not null" json:"storeId"`
	ReviewItemID uint      `gorm:"index" json:"reviewItemId"`
	NFCTagID     uint      `gorm:"index" json:"nfcTagId"`
	SessionID    string    `gorm:"size:128;index;not null" json:"sessionId"`
	ActionType   string    `gorm:"size:64;index:idx_review_logs_store_action_created,priority:2;not null" json:"actionType"`
	PlatformCode string    `gorm:"size:64" json:"platformCode"`
	ClientIP     string    `gorm:"size:64" json:"clientIp"`
	UserAgent    string    `gorm:"size:255" json:"userAgent"`
	CreatedAt    time.Time `gorm:"index:idx_review_logs_store_action_created,priority:3" json:"createdAt"`
}

type ReviewFeedback struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	StoreID         uint      `gorm:"index;not null" json:"storeId"`
	ReviewItemID    uint      `gorm:"index;not null" json:"reviewItemId"`
	SessionID       string    `gorm:"size:128;index;not null" json:"sessionId"`
	PlatformCode    string    `gorm:"size:64;not null" json:"platformCode"`
	FeedbackType    string    `gorm:"size:32;not null" json:"feedbackType"`
	SourceAction    string    `gorm:"size:64;not null" json:"sourceAction"`
	ContentSnapshot string    `gorm:"type:text;not null" json:"contentSnapshot"`
	EditedContent   string    `gorm:"type:text" json:"editedContent"`
	ClientIP        string    `gorm:"size:64" json:"clientIp"`
	UserAgent       string    `gorm:"size:255" json:"userAgent"`
	CreatedAt       time.Time `json:"createdAt"`
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

type StoreGenerationPreference struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	StoreID             uint      `gorm:"uniqueIndex;not null" json:"storeId"`
	FocusKeywords       string    `gorm:"type:json;not null" json:"-"`
	StyleCodes          string    `gorm:"type:json;not null" json:"-"`
	DiversityDimensions string    `gorm:"type:json;not null" json:"-"`
	ReferenceReviews    string    `gorm:"type:json;not null" json:"-"`
	LengthVariance      string    `gorm:"size:32;not null;default:'wide'" json:"lengthVariance"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type NFCTag struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	TagCode string `gorm:"size:128;uniqueIndex;not null" json:"tagCode"`
	// 未绑定门店时为 NULL（指针），避免写入 store_id=0 触发 fk_nfc_store 外键约束。
	StoreID      *uint     `gorm:"index" json:"storeId"`
	LandingToken string    `gorm:"size:128" json:"landingToken"` // 历史字段，落地已改用 store.uuid
	Status       string    `gorm:"size:32;default:'unbound';not null" json:"status"`
	Remark       string    `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// StoreIDValue 返回绑定的门店 ID，未绑定时返回 0。
func (t NFCTag) StoreIDValue() uint {
	if t.StoreID != nil {
		return *t.StoreID
	}
	return 0
}

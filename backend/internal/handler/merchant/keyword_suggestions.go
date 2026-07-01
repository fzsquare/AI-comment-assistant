package merchant

import (
	"strings"

	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// 行业 → 推荐标签。商家一键添加后即成为顾客落地页可选的 chips，
// 从而保证不同店面的标签是各自行业专属的（饭店谈菜品、美甲谈款式…）。
type industryTagSet struct {
	aliases []string
	tags    []string
}

// 顺序与 Python industries 一致：专门行业在前，餐饮兜底；PET 在 BEAUTY 前避免「宠物美容」误配。
var suggestionTable = []industryTagSet{
	{[]string{"足疗", "足浴", "按摩", "推拿", "采耳", "养生", "汗蒸", "理疗"},
		[]string{"手法专业", "力度合适", "技师专业", "服务热情", "环境安静", "干净卫生"}},
	{[]string{"理发", "美发", "发型", "剪发", "烫染", "造型", "美容美发", "发廊"},
		[]string{"发型师专业", "听需求", "剪得满意", "不推办卡", "洗头舒服", "环境好"}},
	{[]string{"美甲", "美睫", "美容美甲", "光疗"},
		[]string{"款式好看", "手法细致", "卸甲不伤", "持久度好", "环境干净", "服务热情"}},
	{[]string{"宠物", "狗", "猫", "宠物美容", "宠物医院", "犬", "铲屎"},
		[]string{"师傅专业", "温柔耐心", "剪得好看", "干净无异味", "宠物不抗拒"}},
	{[]string{"美容", "护肤", "皮肤管理", "面部", "祛痘", "纹绣", "美肤"},
		[]string{"手法专业", "项目效果", "不硬推卡", "环境干净", "服务贴心"}},
	{[]string{"健身", "瑜伽", "私教", "普拉提", "拳击", "舞蹈", "游泳", "动感单车"},
		[]string{"教练专业", "纠正动作", "器械齐全", "不推私教", "环境好", "卫生好"}},
	{[]string{"ktv", "酒吧", "桌游", "剧本杀", "密室", "电玩", "网咖", "棋牌", "台球", "轰趴", "娱乐"},
		[]string{"包厢大", "设备好", "隔音好", "服务热情", "性价比高", "环境好"}},
	{[]string{"洗车", "汽车", "保养", "贴膜", "镀晶", "维修", "补胎", "改装", "养护"},
		[]string{"洗得干净", "师傅仔细", "报价透明", "效率高", "环境好"}},
	{[]string{"餐", "美食", "菜", "火锅", "烧烤", "烤肉", "饭", "小吃", "咖啡", "茶", "面", "甜品", "烘焙"},
		[]string{"味道好", "招牌菜", "分量足", "适合聚餐", "服务热情", "环境舒服", "性价比高"}},
}

var defaultSuggestionTags = []string{"服务热情", "环境舒服", "性价比高", "体验好", "干净卫生"}

func suggestTags(industryType string) []string {
	t := strings.ToLower(strings.TrimSpace(industryType))
	if t != "" {
		for _, set := range suggestionTable {
			for _, alias := range set.aliases {
				if strings.Contains(t, strings.ToLower(alias)) {
					return set.tags
				}
			}
		}
	}
	return defaultSuggestionTags
}

// keywordSuggestions 返回当前商家门店行业对应的推荐标签。
func (h *Handler) keywordSuggestions(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Success(c, gin.H{"tags": defaultSuggestionTags})
		return
	}
	response.Success(c, gin.H{"tags": suggestTags(store.IndustryType)})
}

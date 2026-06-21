"""HTTP 请求/响应模型。Go 后端通过这个契约调用，契约里不含任何 provider 字眼。"""
from __future__ import annotations

from typing import Annotated, List, Literal

from pydantic import BaseModel, Field

PlatformCode = Literal["dianping", "xiaohongshu", "douyin"]
SatisfactionLevel = Literal["非常满意", "比较满意", "一般", "有点失望", "非常失望"]
Grade = Literal["", "S", "A", "B", "C", "D"]
Keyword = Annotated[str, Field(min_length=1, max_length=40)]
Tag = Annotated[str, Field(min_length=1, max_length=40)]


class StoreContext(BaseModel):
    store_name: str = Field(min_length=1, max_length=120)
    industry_type: str = Field(default="", max_length=80)
    store_intro: str = Field(default="", max_length=1000)
    brand_tone: str = Field(default="", max_length=120)
    address: str = Field(default="", max_length=200)


class GenerateRequest(BaseModel):
    store: StoreContext
    # 菜品/场景标签来源（复用 Go 侧的 StoreKeyword）。生成时据此打 tag，
    # 也是“顾客选了什么 → 即时取池中评价”的过滤依据。
    keywords: List[Keyword] = Field(default_factory=list, max_length=20)
    platform: PlatformCode = "dianping"
    count: int = Field(default=1, ge=1, le=50)
    # 默认“比较满意”——辅助真实到店顾客发真实好评，不是极端吹捧。
    satisfaction: SatisfactionLevel = "比较满意"


class ReviewItem(BaseModel):
    content: str = Field(min_length=1, max_length=2000)
    tags: List[Tag] = Field(default_factory=list, max_length=10)
    score: int = Field(default=0, ge=0, le=100)
    grade: Grade = ""  # S / A / B / C / D（约束手册 6.1）
    revisions: int = Field(default=0, ge=0)  # 经过几轮重写才达标


class GenerateResponse(BaseModel):
    platform: PlatformCode
    requested: int = Field(ge=1, le=50)
    produced: int = Field(ge=0, le=50)
    items: List[ReviewItem] = Field(default_factory=list, max_length=50)

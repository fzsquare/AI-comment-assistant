INSERT INTO admin_users (account, password_hash, name, status)
VALUES ('admin', '$2a$10$hokEuqVp1VwxjW4/2dzt2OzWNbWfBA6t9V0rgt7pSYOU3hAr9oAE6', '平台管理员', 1)
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO merchant_users (account, password_hash, merchant_name, contact_name, status)
VALUES ('merchant', '$2a$10$hokEuqVp1VwxjW4/2dzt2OzWNbWfBA6t9V0rgt7pSYOU3hAr9oAE6', '示例商家', '张三', 1)
ON DUPLICATE KEY UPDATE merchant_name = VALUES(merchant_name), contact_name = VALUES(contact_name);

INSERT INTO stores (merchant_user_id, store_name, industry_type, store_intro, address, primary_platform_style, brand_tone, status)
SELECT id, '示例餐厅', '餐饮', '一家适合朋友聚会的本地餐厅', '示例路 88 号', 'xiaohongshu', '轻松自然', 1
FROM merchant_users WHERE account = 'merchant'
ON DUPLICATE KEY UPDATE store_name = VALUES(store_name), industry_type = VALUES(industry_type), store_intro = VALUES(store_intro), address = VALUES(address), primary_platform_style = VALUES(primary_platform_style), brand_tone = VALUES(brand_tone), status = VALUES(status);

INSERT INTO store_keywords (store_id, keyword, sort_no)
SELECT s.id, '环境舒服', 1 FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM store_keywords existing WHERE existing.store_id = s.id AND existing.keyword = '环境舒服')
UNION ALL
SELECT s.id, '服务热情', 2 FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM store_keywords existing WHERE existing.store_id = s.id AND existing.keyword = '服务热情')
UNION ALL
SELECT s.id, '适合聚餐', 3 FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM store_keywords existing WHERE existing.store_id = s.id AND existing.keyword = '适合聚餐');

INSERT INTO store_images (store_id, image_url, thumbnail_url, status, sort_no)
SELECT s.id, 'https://images.unsplash.com/photo-1517248135467-4c7edcad34c4', 'https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?w=400', 1, 1
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM store_images existing WHERE existing.store_id = s.id AND existing.image_url = 'https://images.unsplash.com/photo-1517248135467-4c7edcad34c4')
UNION ALL
SELECT s.id, 'https://images.unsplash.com/photo-1552566626-52f8b828add9', 'https://images.unsplash.com/photo-1552566626-52f8b828add9?w=400', 1, 2
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM store_images existing WHERE existing.store_id = s.id AND existing.image_url = 'https://images.unsplash.com/photo-1552566626-52f8b828add9');

INSERT INTO store_platform_links (store_id, platform_code, platform_name, button_text, target_url, backup_url, sort_no, status)
SELECT s.id, 'meituan', '美团', '去美团评论', 'https://www.meituan.com/', 'https://www.meituan.com/', 1, 1
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
ON DUPLICATE KEY UPDATE platform_name = VALUES(platform_name), button_text = VALUES(button_text), target_url = VALUES(target_url), backup_url = VALUES(backup_url), sort_no = VALUES(sort_no), status = VALUES(status);

INSERT INTO store_platform_links (store_id, platform_code, platform_name, button_text, target_url, backup_url, sort_no, status)
SELECT s.id, 'dianping', '大众点评', '去大众点评评论', 'https://www.dianping.com/', 'https://www.dianping.com/', 2, 1
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
ON DUPLICATE KEY UPDATE platform_name = VALUES(platform_name), button_text = VALUES(button_text), target_url = VALUES(target_url), backup_url = VALUES(backup_url), sort_no = VALUES(sort_no), status = VALUES(status);

INSERT INTO nfc_tags (tag_code, store_id, landing_token, status, remark)
SELECT 'TAG-DEMO-001', s.id, 'landing-demo-001', 'bound', '前台标签'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
ON DUPLICATE KEY UPDATE store_id = VALUES(store_id), status = VALUES(status), remark = VALUES(remark);

INSERT INTO review_items (store_id, platform_style, content, source_type, generation_batch_no, is_dispatched, status)
SELECT s.id, 'meituan', '这家店环境很舒服，服务也很自然，整体体验挺不错，下次还会再来。', 'seed', 'seed_batch_meituan', 0, 'available'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM review_items existing WHERE existing.store_id = s.id AND existing.platform_style = 'meituan' AND existing.generation_batch_no = 'seed_batch_meituan' AND existing.content = '这家店环境很舒服，服务也很自然，整体体验挺不错，下次还会再来。')
UNION ALL
SELECT s.id, 'meituan', '和朋友一起过来用餐，出品稳定，环境轻松，适合聚餐聊天。', 'seed', 'seed_batch_meituan', 0, 'available'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM review_items existing WHERE existing.store_id = s.id AND existing.platform_style = 'meituan' AND existing.generation_batch_no = 'seed_batch_meituan' AND existing.content = '和朋友一起过来用餐，出品稳定，环境轻松，适合聚餐聊天。')
UNION ALL
SELECT s.id, 'meituan', '店里整体氛围不错，服务挺热情，第一次来体验就感觉很好。', 'seed', 'seed_batch_meituan', 0, 'available'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM review_items existing WHERE existing.store_id = s.id AND existing.platform_style = 'meituan' AND existing.generation_batch_no = 'seed_batch_meituan' AND existing.content = '店里整体氛围不错，服务挺热情，第一次来体验就感觉很好。')
UNION ALL
SELECT s.id, 'dianping', '环境和服务都很在线，点评过来体验没有踩雷，整体挺满意。', 'seed', 'seed_batch_dianping', 0, 'available'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM review_items existing WHERE existing.store_id = s.id AND existing.platform_style = 'dianping' AND existing.generation_batch_no = 'seed_batch_dianping' AND existing.content = '环境和服务都很在线，点评过来体验没有踩雷，整体挺满意。')
UNION ALL
SELECT s.id, 'dianping', '朋友推荐来的，菜品和环境都比较稳定，适合周末聚餐。', 'seed', 'seed_batch_dianping', 0, 'available'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM review_items existing WHERE existing.store_id = s.id AND existing.platform_style = 'dianping' AND existing.generation_batch_no = 'seed_batch_dianping' AND existing.content = '朋友推荐来的，菜品和环境都比较稳定，适合周末聚餐。')
UNION ALL
SELECT s.id, 'dianping', '店员服务挺热情，整体体验自然舒服，之后还会考虑再来。', 'seed', 'seed_batch_dianping', 0, 'available'
FROM stores s JOIN merchant_users m ON s.merchant_user_id = m.id WHERE m.account = 'merchant'
AND NOT EXISTS (SELECT 1 FROM review_items existing WHERE existing.store_id = s.id AND existing.platform_style = 'dianping' AND existing.generation_batch_no = 'seed_batch_dianping' AND existing.content = '店员服务挺热情，整体体验自然舒服，之后还会考虑再来。');

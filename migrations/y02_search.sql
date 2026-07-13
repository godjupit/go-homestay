USE `looklook_travel`;
SET NAMES utf8mb4;

ALTER TABLE `homestay`
  ADD COLUMN `city` varchar(64) NOT NULL DEFAULT '' COMMENT '城市' AFTER `info`,
  ADD COLUMN `tags` varchar(255) NOT NULL DEFAULT '' COMMENT '逗号分隔标签' AFTER `city`,
  ADD COLUMN `star` decimal(2,1) NOT NULL DEFAULT 0.0 COMMENT '综合评分' AFTER `tags`,
  ADD COLUMN `latitude` decimal(10,7) NOT NULL DEFAULT 0 COMMENT '纬度' AFTER `star`,
  ADD COLUMN `longitude` decimal(10,7) NOT NULL DEFAULT 0 COMMENT '经度' AFTER `latitude`,
  ADD KEY `idx_homestay_city_price` (`city`,`homestay_price`),
  ADD KEY `idx_homestay_star` (`star`);

CREATE TABLE IF NOT EXISTS `search_event_outbox` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `event_key` varchar(128) NOT NULL,
  `aggregate_id` bigint NOT NULL,
  `event_type` varchar(32) NOT NULL DEFAULT 'upsert',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=待同步,1=已完成',
  `retry_count` int NOT NULL DEFAULT 0,
  `next_retry_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_error` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `published_at` datetime NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_search_event_key` (`event_key`),
  KEY `idx_search_outbox_pending` (`status`,`next_retry_at`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

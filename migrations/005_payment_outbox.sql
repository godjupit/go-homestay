CREATE DATABASE IF NOT EXISTS `looklook_payment` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `looklook_payment`;

CREATE TABLE IF NOT EXISTS `event_outbox` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `event_key` varchar(100) NOT NULL,
  `topic` varchar(128) NOT NULL,
  `message_key` varchar(128) NOT NULL,
  `payload` json NOT NULL,
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=pending,1=published',
  `retry_count` int NOT NULL DEFAULT 0,
  `next_retry_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `published_at` datetime NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_key` (`event_key`),
  KEY `idx_pending` (`status`,`next_retry_at`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

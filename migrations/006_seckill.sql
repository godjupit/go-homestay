CREATE DATABASE IF NOT EXISTS `looklook_order` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `looklook_order`;

CREATE TABLE IF NOT EXISTS `seckill_activity` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `homestay_id` bigint NOT NULL,
  `title` varchar(128) NOT NULL,
  `price` bigint NOT NULL COMMENT '秒杀单晚价格，单位分',
  `stock` int NOT NULL,
  `sold_count` int NOT NULL DEFAULT 0,
  `start_time` datetime NOT NULL,
  `end_time` datetime NOT NULL,
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '0=停用,1=启用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_activity_window` (`status`,`start_time`,`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `seckill_order` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `reservation_sn` char(25) NOT NULL,
  `activity_id` bigint NOT NULL,
  `user_id` bigint NOT NULL,
  `order_sn` char(25) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=处理中,1=已创建',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_reservation_sn` (`reservation_sn`),
  UNIQUE KEY `uk_activity_user` (`activity_id`,`user_id`),
  KEY `idx_order_sn` (`order_sn`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

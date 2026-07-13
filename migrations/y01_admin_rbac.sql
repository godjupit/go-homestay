CREATE DATABASE IF NOT EXISTS `looklook_usercenter` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `looklook_usercenter`;
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `admin_user` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(64) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `nickname` varchar(64) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 1,
  `business_id` bigint NOT NULL DEFAULT 0 COMMENT '所属商家数据范围',
  `linked_user_id` bigint NOT NULL DEFAULT 0 COMMENT '关联业务用户，用于本人数据范围',
  `version` bigint NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `admin_role` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `name` varchar(64) NOT NULL,
  `status` tinyint NOT NULL DEFAULT 1,
  `scope_type` tinyint NOT NULL DEFAULT 4 COMMENT '1=全部,2=所属商家,3=自定义商家,4=本人',
  `version` bigint NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `admin_permission` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` varchar(100) NOT NULL,
  `name` varchar(100) NOT NULL,
  `method` varchar(10) NOT NULL DEFAULT 'POST',
  `path` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_permission_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `admin_user_role` (
  `admin_user_id` bigint NOT NULL,
  `role_id` bigint NOT NULL,
  PRIMARY KEY (`admin_user_id`,`role_id`),
  KEY `idx_user_role_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `admin_role_permission` (
  `role_id` bigint NOT NULL,
  `permission_id` bigint NOT NULL,
  PRIMARY KEY (`role_id`,`permission_id`),
  KEY `idx_role_permission_permission` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `admin_role_data_scope` (
  `role_id` bigint NOT NULL,
  `business_id` bigint NOT NULL,
  PRIMARY KEY (`role_id`,`business_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `admin_audit_log` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `admin_user_id` bigint NOT NULL DEFAULT 0,
  `username` varchar(64) NOT NULL DEFAULT '',
  `permission_code` varchar(100) NOT NULL DEFAULT '',
  `method` varchar(10) NOT NULL,
  `path` varchar(255) NOT NULL,
  `request_id` varchar(64) NOT NULL DEFAULT '',
  `ip` varchar(64) NOT NULL DEFAULT '',
  `http_status` int NOT NULL DEFAULT 0,
  `success` tinyint NOT NULL DEFAULT 0,
  `duration_ms` bigint NOT NULL DEFAULT 0,
  `request_body` text,
  `error_message` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_audit_admin_time` (`admin_user_id`,`created_at`),
  KEY `idx_audit_permission_time` (`permission_code`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT IGNORE INTO `admin_role` (`id`,`code`,`name`,`status`,`scope_type`) VALUES (1,'super_admin','超级管理员',1,1);
INSERT IGNORE INTO `admin_permission` (`id`,`code`,`name`,`method`,`path`) VALUES
  (1,'admin:user:list','管理员列表','POST','/admin/v1/user/list'),
  (2,'admin:user:create','创建管理员','POST','/admin/v1/user/create'),
  (3,'admin:user:update','更新管理员','POST','/admin/v1/user/update'),
  (4,'admin:user:assign','分配管理员角色','POST','/admin/v1/user/assignRoles'),
  (5,'admin:role:list','角色列表','POST','/admin/v1/role/list'),
  (6,'admin:role:create','创建角色','POST','/admin/v1/role/create'),
  (7,'admin:role:configure','配置角色','POST','/admin/v1/role/configure'),
  (8,'admin:permission:list','权限列表','POST','/admin/v1/permission/list'),
  (9,'admin:permission:create','创建权限','POST','/admin/v1/permission/create'),
  (10,'admin:audit:list','操作审计列表','POST','/admin/v1/audit/list'),
  (11,'travel:homestay:list','管理端民宿列表','POST','/admin/v1/homestay/list'),
  (12,'travel:homestay:update','更新民宿','POST','/admin/v1/homestay/update'),
  (13,'search:index:rebuild','重建搜索索引','POST','/admin/v1/search/rebuild');
INSERT IGNORE INTO `admin_role_permission` (`role_id`,`permission_id`) SELECT 1,id FROM `admin_permission`;

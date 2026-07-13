SET NAMES utf8mb4;
USE `looklook_usercenter`;
INSERT IGNORE INTO `user` (`id`,`delete_time`,`mobile`,`password`,`nickname`,`sex`,`avatar`,`info`)
VALUES (1,FROM_UNIXTIME(0),'18888888888','e10adc3949ba59abbe56e057f20f883e','Gin LookLook 房东',0,'','模块化单体演示账号');
INSERT IGNORE INTO `user_auth` (`id`,`delete_time`,`user_id`,`auth_key`,`auth_type`)
VALUES (1,FROM_UNIXTIME(0),1,'18888888888','system');

USE `looklook_travel`;
INSERT IGNORE INTO `homestay_business` (`id`,`delete_time`,`title`,`user_id`,`info`,`boss_info`,`row_state`,`star`,`tags`,`cover`,`header_img`)
VALUES (1,FROM_UNIXTIME(0),'LookLook 精选民宿',1,'面向面试演示的民宿商家','热情、可靠的房东',1,4.9,'安静舒适','','');
INSERT IGNORE INTO `homestay` (`id`,`delete_time`,`title`,`sub_title`,`banner`,`info`,`people_num`,`homestay_business_id`,`user_id`,`row_state`,`row_type`,`food_info`,`food_price`,`homestay_price`,`market_homestay_price`)
VALUES (11,FROM_UNIXTIME(0),'Interview Demo Homestay','Gin 模块化单体演示房源','','包含订单、支付、延迟任务和可观测性完整链路',4,1,1,1,0,'双人早餐',3000,29900,39900);
UPDATE `homestay` SET `city`='杭州',`tags`='西湖,亲子,早餐',`star`=4.8,`latitude`=30.2525010,`longitude`=120.1650240 WHERE `id`=11;
INSERT IGNORE INTO `homestay_activity` (`id`,`delete_time`,`row_type`,`data_id`,`row_status`)
VALUES (1,FROM_UNIXTIME(0),'preferredHomestay',11,1),(2,FROM_UNIXTIME(0),'goodBusiness',1,1);
INSERT IGNORE INTO `homestay_comment` (`id`,`delete_time`,`homestay_id`,`user_id`,`content`,`star`)
VALUES (1,FROM_UNIXTIME(0),11,1,'环境很好，业务链路也很完整',JSON_OBJECT('clean',5,'service',4.8,'location',4.7));

USE `looklook_order`;
INSERT IGNORE INTO `seckill_activity` (`id`,`homestay_id`,`title`,`price`,`stock`,`sold_count`,`start_time`,`end_time`,`status`)
VALUES (1,11,'Gin LookLook 限时秒杀',9900,20,0,DATE_SUB(NOW(),INTERVAL 1 DAY),DATE_ADD(NOW(),INTERVAL 365 DAY),1);

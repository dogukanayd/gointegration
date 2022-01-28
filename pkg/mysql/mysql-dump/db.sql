DROP DATABASE IF EXISTS slaDB;
CREATE DATABASE slaDB;

USE slaDB;

DROP TABLE IF EXISTS `business_slas`;

CREATE TABLE IF NOT EXISTS `business_slas`
(
    `id`           int(11)   NOT NULL AUTO_INCREMENT,
    `product`      varchar(255),
    `feature`      varchar(500),
    `partner_name` varchar(500),
    `phase`        varchar(50),
    `count`        int,
    `unit_id`      int(11),
    `unit_name`    varchar(100),
    `started_at`   int(11),
    `completed_at` int(11),
    `created_at`   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `unique_record` (`product`, `partner_name`, `unit_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  AUTO_INCREMENT = 1;



-- up
CREATE TABLE `rbacassignment` (
	`item_name` VARCHAR(64) NOT NULL,
	`user_id` VARCHAR(64) NOT NULL,
	`created_at` DATETIME NOT NULL,
	PRIMARY KEY (`item_name`, `user_id`)
);

CREATE TABLE `rbacrule` (
	`name` VARCHAR(64) NOT NULL,
	`data` TEXT NULL,
	`created_at` DATETIME NOT NULL,
	`updated_at` DATETIME NULL,
	PRIMARY KEY (`name`)
);

CREATE TABLE `rbacitem` (
	`name` VARCHAR(64) NOT NULL,
	`type` INT NOT NULL,
	`description` TEXT NULL,
	`rule_name` VARCHAR(64) NULL,
	`data` TEXT NULL,
	`created_at` DATETIME NOT NULL,
	`updated_at` DATETIME NULL,
	PRIMARY KEY (`name`)
);

CREATE TABLE `rbacitemchild` (
	`parent` VARCHAR(64) NOT NULL,
	`child` VARCHAR(64) NOT NULL,
	PRIMARY KEY (`parent`, `child`)
);

ALTER TABLE `rbacassignment` ADD CONSTRAINT `item_name_fk` FOREIGN KEY (item_name) REFERENCES `rbacitem`(name) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE `rbacitem` ADD CONSTRAINT `rule_name_fk` FOREIGN KEY (rule_name) REFERENCES `rbacrule`(name) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE `rbacitemchild` ADD CONSTRAINT `parent_fk` FOREIGN KEY (parent) REFERENCES `rbacitem`(name) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE `rbacitemchild` ADD CONSTRAINT `child_fk` FOREIGN KEY (child) REFERENCES `rbacitem`(name) ON DELETE CASCADE ON UPDATE CASCADE;

-- down
ALTER TABLE `rbacitemchild` DROP FOREIGN KEY `child_fk`;
ALTER TABLE `rbacitemchild` DROP FOREIGN KEY `parent_fk`;
ALTER TABLE `rbacitem` DROP FOREIGN KEY `rule_name_fk`;
ALTER TABLE `rbacassignment` DROP FOREIGN KEY `item_name_fk`;

DROP TABLE `rbacitemchild`;
DROP TABLE `rbacitem`;
DROP TABLE `rbacrule`;
DROP TABLE `rbacassignment`;
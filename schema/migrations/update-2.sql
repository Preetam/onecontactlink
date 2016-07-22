DROP TABLE `tokens`;

ALTER TABLE `emails` ADD COLUMN `status` tinyint(3) unsigned NOT NULL DEFAULT '0' AFTER `user`;

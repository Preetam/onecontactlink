ALTER TABLE `requests` ADD COLUMN `email_sent` int(10) unsigned NOT NULL DEFAULT '0' AFTER `status`;
INSERT INTO `schema_version` VALUES (2, UNIX_TIMESTAMP());

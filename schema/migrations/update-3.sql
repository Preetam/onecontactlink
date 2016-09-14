ALTER TABLE `emails` ADD UNIQUE KEY `address_deleted` (`address`, `deleted`);
ALTER TABLE `emails` DROP KEY `address`;

CREATE TABLE `emails` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `address` varchar(255) NOT NULL,
  `user` int(10) unsigned NOT NULL DEFAULT '0',
  `created` int(10) unsigned NOT NULL,
  `updated` int(10) unsigned NOT NULL,
  `deleted` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`,`address`),
  UNIQUE KEY `address` (`address`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=latin1;

CREATE TABLE `schema_version` (
  `version` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ts` int(10) unsigned NOT NULL,
  PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `tokens` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user` int(10) NOT NULL DEFAULT '0',
  `value` char(32) DEFAULT NULL,
  `created` int(10) unsigned NOT NULL,
  `updated` int(10) unsigned NOT NULL,
  `deleted` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_deleted` (`user`,`deleted`),
  UNIQUE KEY `value` (`value`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=latin1;

CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(60) CHARACTER SET utf8 NOT NULL,
  `main_email` int(10) NOT NULL,
  `created` int(10) unsigned NOT NULL,
  `updated` int(10) unsigned NOT NULL,
  `deleted` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `main_email` (`main_email`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=latin1;


INSERT INTO `schema_version` VALUES
  (1, UNIX_TIMESTAMP());

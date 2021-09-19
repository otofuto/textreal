create table `docs` (
    `id` int auto_increment,
    `title` varchar(255) not null default '',
    `text` text not null,
    `token` varchar(255) not null,
    primary key (`id`)
);
create table `docs` (
    `id` int auto_increment,
    `text` text not null,
    `token` varchar(255) not null,
    primary key (`id`)
);